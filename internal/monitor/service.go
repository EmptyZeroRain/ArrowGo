package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"monitor/internal/database"
	"monitor/internal/elasticsearch"
	"monitor/internal/logger"
	"monitor/internal/models"

	"go.uber.org/zap"
)

type Service struct {
	targets   map[uint32]*MonitorTarget
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	es        *elasticsearch.Client

	// Worker pool for high concurrency
	checkQueue chan *MonitorTarget
	workerPool int32
	wg         sync.WaitGroup

	// Async ES writes
	esBuffer chan *esWriteTask
}

type esWriteTask struct {
	target *MonitorTarget
	result *CheckResult
}

func NewService(esClient *elasticsearch.Client) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	// Get worker count from config (default 100 workers)
	workerCount := int32(100)

	s := &Service{
		targets:    make(map[uint32]*MonitorTarget),
		ctx:        ctx,
		cancel:     cancel,
		es:         esClient,
		checkQueue: make(chan *MonitorTarget, 1000), // Buffered queue
		workerPool: workerCount,
		esBuffer:   make(chan *esWriteTask, 500), // Buffer for ES writes
	}

	// Start worker pool
	s.startWorkerPool()

	// Start async ES writer
	s.startAsyncESWriter()

	return s
}

func (s *Service) AddTarget(target *MonitorTarget) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.targets[target.ID] = target
	go s.monitorTarget(target)

	return nil
}

// TriggerCheck manually triggers an immediate check for a target
func (s *Service) TriggerCheck(targetID uint32) error {
	s.mu.RLock()
	target, exists := s.targets[targetID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("target not found")
	}

	// Trigger check in background
	go s.checkTarget(target)

	return nil
}

// startWorkerPool starts the worker pool for concurrent checks
func (s *Service) startWorkerPool() {
	logger.Info("Starting worker pool", zap.Int32("workers", s.workerPool))

	for i := int32(0); i < s.workerPool; i++ {
		s.wg.Add(1)
		go func(workerID int32) {
			defer s.wg.Done()
			s.checkWorker(workerID)
		}(i)
	}
}

// checkWorker processes checks from the queue
func (s *Service) checkWorker(workerID int32) {
	for {
		select {
		case <-s.ctx.Done():
			return
		case target := <-s.checkQueue:
			s.checkTarget(target)
		}
	}
}

// startAsyncESWriter starts the async Elasticsearch writer
func (s *Service) startAsyncESWriter() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.esWriter()
	}()
}

// esWriter processes ES writes asynchronously
func (s *Service) esWriter() {
	for {
		select {
		case <-s.ctx.Done():
			// Flush remaining writes
			for task := range s.esBuffer {
				s.writeToElasticsearch(task.target, task.result)
			}
			return
		case task := <-s.esBuffer:
			s.writeToElasticsearch(task.target, task.result)
		}
	}
}

func (s *Service) RemoveTarget(id uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.targets[id]; exists {
		delete(s.targets, id)
		return nil
	}
	return fmt.Errorf("target not found")
}

func (s *Service) GetTarget(id uint32) (*MonitorTarget, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	target, exists := s.targets[id]
	if !exists {
		return nil, fmt.Errorf("target not found")
	}
	return target, nil
}

func (s *Service) ListTargets() []*MonitorTarget {
	s.mu.RLock()
	defer s.mu.RUnlock()

	targets := make([]*MonitorTarget, 0, len(s.targets))
	for _, target := range s.targets {
		targets = append(targets, target)
	}
	return targets
}

func (s *Service) monitorTarget(target *MonitorTarget) {
	ticker := time.NewTicker(time.Duration(target.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// Send to worker pool queue instead of executing directly
			select {
			case s.checkQueue <- target:
				// Successfully queued
			default:
				// Queue full, log warning and skip this check
				logger.Warn("Check queue full, skipping check",
					zap.Uint32("target_id", target.ID),
					zap.String("target_name", target.Name))
			}
		}
	}
}

func (s *Service) checkTarget(target *MonitorTarget) {
	checker, err := NewChecker(target.Type)
	if err != nil {
		log.Printf("Failed to create checker for target %d: %v", target.ID, err)
		return
	}

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	result, err := checker.Check(ctx, target)
	if err != nil {
		log.Printf("Check failed for target %d: %v", target.ID, err)
		return
	}

	s.saveResult(target, result)
}

func (s *Service) saveResult(target *MonitorTarget, result *CheckResult) {
	db := database.GetDB()

	var status models.MonitorStatus
	err := db.Where("target_id = ?", target.ID).First(&status).Error
	if err != nil {
		status = models.MonitorStatus{
			TargetID: target.ID,
		}
	}

	status.Status = result.Status
	status.ResponseTime = result.ResponseTime
	status.Message = result.Message
	status.CheckedAt = time.Now()

	// Save SSL certificate info if available (for HTTPS, SSL, TLS)
	if target.Type == "https" || target.Type == "ssl" || target.Type == "tls" {
		if daysUntilExpiry, ok := result.Response.Headers["days_until_expiry"]; ok {
			var days int
			if _, err := fmt.Sscanf(daysUntilExpiry, "%d", &days); err == nil {
				status.SSLDaysUntilExpiry = &days
			}
		}
		if issuer, ok := result.Response.Headers["issuer"]; ok {
			status.SSLIssuer = &issuer
		}
		if subject, ok := result.Response.Headers["subject"]; ok {
			status.SSLSubject = &subject
		}
		if serial, ok := result.Response.Headers["serial"]; ok {
			status.SSLEserial = &serial
		}
	}

	// Save resolved IP if available
	if resolvedIP, ok := result.Response.Headers["resolved_ip"]; ok {
		status.ResolvedIP = &resolvedIP
	} else if resolvedIP, ok := result.Response.Headers["ip"]; ok {
		status.ResolvedIP = &resolvedIP
	}

	// Save full check result data as JSON
	if len(result.Data) > 0 {
		dataJSON, err := json.Marshal(result.Data)
		if err == nil {
			dataStr := string(dataJSON)
			status.Data = &dataStr
		}
	}

	// Save DNS records if available
	if target.Type == "dns" && result.Response.Body != "" {
		dnsRecords := result.Response.Body
		status.DNSRecords = &dnsRecords
	}

	history := models.MonitorHistory{
		TargetID:     target.ID,
		Status:       result.Status,
		ResponseTime: result.ResponseTime,
		Message:      result.Message,
		CheckedAt:    time.Now(),
	}

	if err := db.Save(&status).Error; err != nil {
		log.Printf("Failed to save status for target %d: %v", target.ID, err)
	}

	if err := db.Create(&history).Error; err != nil {
		log.Printf("Failed to save history for target %d: %v", target.ID, err)
	}

	s.updateUptimePercentage(target.ID)

	// Async save to Elasticsearch
	select {
	case s.esBuffer <- &esWriteTask{target: target, result: result}:
		// Successfully queued for ES write
	default:
		// Buffer full, log warning but don't block
		logger.Warn("ES buffer full, dropping log",
			zap.Uint32("target_id", target.ID))
	}

	// Always write to file log (non-blocking, independent of ES)
	s.writeFileLog(target, result)
}

// writeToElasticsearch actually writes to ES
func (s *Service) writeToElasticsearch(target *MonitorTarget, result *CheckResult) {
	if s.es == nil {
		return // ES 未启用
	}

	// 构建 ES 日志条目
	entry := &elasticsearch.LogEntry{
		TargetID:     target.ID,
		TargetName:   target.Name,
		TargetType:   target.Type,
		Address:      target.Address,
		Status:       result.Status,
		ResponseTime: result.ResponseTime,
		Message:      result.Message,
	}

	// 填充请求信息
	entry.Request.Method = result.Request.Method
	entry.Request.ResolvedURL = result.Request.URL
	entry.Request.Headers = result.Request.Headers
	entry.Request.Body = result.Request.Body

	// 填充响应信息
	if result.Response.StatusCode != 0 {
		entry.Response.StatusCode = result.Response.StatusCode
	}
	entry.Response.Headers = result.Response.Headers
	entry.Response.Body = result.Response.Body
	entry.Response.ContentLength = result.Response.ContentLength

	// 填充错误信息
	if result.Error != nil {
		entry.Error.Type = result.Error.Type
		entry.Error.Message = result.Error.Message
	}

	// 索引到 ES
	if err := s.es.IndexLog(entry); err != nil {
		logger.Log.Error(fmt.Sprintf("Failed to index log to ES: target_id=%d, error=%v",
			target.ID, err))
	}
}

// writeFileLog writes check result to file-based log
func (s *Service) writeFileLog(target *MonitorTarget, result *CheckResult) {
	entry := &logger.CheckLogEntry{
		TargetID:     int(target.ID),
		TargetName:   target.Name,
		Type:         target.Type,
		Address:      target.Address,
		Status:       result.Status,
		ResponseTime: result.ResponseTime,
		Message:      result.Message,
	}

	// Add request details if available
	if result.Request.Method != "" || result.Request.URL != "" {
		entry.Request = make(map[string]interface{})
		if result.Request.Method != "" {
			entry.Request["method"] = result.Request.Method
		}
		if result.Request.URL != "" {
			entry.Request["url"] = result.Request.URL
		}
		// Always save headers, even if empty (to show what was sent)
		entry.Request["headers"] = result.Request.Headers
		if result.Request.Body != "" {
			entry.Request["body"] = result.Request.Body
		}
	}

	// Add response details if available
	if result.Response.StatusCode != 0 || len(result.Response.Headers) > 0 {
		entry.Response = make(map[string]interface{})
		if result.Response.StatusCode != 0 {
			entry.Response["status_code"] = result.Response.StatusCode
		}
		if len(result.Response.Headers) > 0 {
			entry.Response["headers"] = result.Response.Headers
		}
		// Don't save body content, only size
		if result.Response.Body != "" {
			entry.Response["body_size"] = len(result.Response.Body)
		}
		if result.Response.ContentLength > 0 {
			entry.Response["content_length"] = result.Response.ContentLength
		}
	}

	// Add error details if available
	if result.Error != nil {
		if entry.Request == nil {
			entry.Request = make(map[string]interface{})
		}
		entry.Request["error"] = map[string]interface{}{
			"type":    result.Error.Type,
			"message": result.Error.Message,
		}
	}

	// Write to log file
	if err := logger.WriteCheckLog("logs", entry); err != nil {
		logger.Log.Warn("Failed to write check log to file",
			zap.Int("target_id", int(target.ID)),
			zap.Error(err),
		)
	}
}

func (s *Service) updateUptimePercentage(targetID uint32) {
	db := database.GetDB()

	var historyCount int64
	var upCount int64

	db.Model(&models.MonitorHistory{}).
		Where("target_id = ? AND checked_at >= ?", targetID, time.Now().AddDate(0, 0, -30)).
		Count(&historyCount)

	db.Model(&models.MonitorHistory{}).
		Where("target_id = ? AND status = ? AND checked_at >= ?", targetID, "up", time.Now().AddDate(0, 0, -30)).
		Count(&upCount)

	var status models.MonitorStatus
	if err := db.Where("target_id = ?", targetID).First(&status).Error; err == nil {
		if historyCount > 0 {
			status.UptimePercentage = int32((upCount * 100) / historyCount)
		} else {
			status.UptimePercentage = 0
		}
		db.Save(&status)
	}
}

func (s *Service) LoadTargetsFromDB() error {
	db := database.GetDB()

	var dbTargets []models.MonitorTarget
	if err := db.Find(&dbTargets).Error; err != nil {
		return err
	}

	for _, dbTarget := range dbTargets {
		if !dbTarget.Enabled {
			continue
		}

		var metadata map[string]string
		if dbTarget.Metadata != "" {
			if err := json.Unmarshal([]byte(dbTarget.Metadata), &metadata); err != nil {
				metadata = make(map[string]string)
			}
		}

		var httpHeaders map[string]string
		if dbTarget.HTTPHeaders != "" {
			if err := json.Unmarshal([]byte(dbTarget.HTTPHeaders), &httpHeaders); err != nil {
				httpHeaders = make(map[string]string)
			}
		}

		// Parse expected status codes
		var expectedStatusCodes []int
		if dbTarget.ExpectedStatusCodes != "" {
			codesStr := strings.Split(dbTarget.ExpectedStatusCodes, ",")
			for _, codeStr := range codesStr {
				codeStr = strings.TrimSpace(codeStr)
				if codeStr != "" {
					var code int
					if _, err := fmt.Sscanf(codeStr, "%d", &code); err == nil {
						expectedStatusCodes = append(expectedStatusCodes, code)
					}
				}
			}
		}

		target := &MonitorTarget{
			ID:       dbTarget.ID,
			Name:     dbTarget.Name,
			Type:     dbTarget.Type,
			Address:  dbTarget.Address,
			Port:     dbTarget.Port,
			Interval: dbTarget.Interval,
			Metadata: metadata,
			Enabled:  dbTarget.Enabled,
			// HTTP/HTTPS specific fields
			HTTPMethod:          dbTarget.HTTPMethod,
			HTTPHeaders:         httpHeaders,
			HTTPBody:            dbTarget.HTTPBody,
			ResolvedHost:        dbTarget.ResolvedHost,
			FollowRedirects:     dbTarget.FollowRedirects,
			MaxRedirects:        dbTarget.MaxRedirects,
			ExpectedStatusCodes: expectedStatusCodes,
			// DNS specific fields
			DNSServer: dbTarget.DNSServer,
			// SSL/TLS specific fields
			SSLWarnDays:    dbTarget.SSLWarnDays,
			SSLCriticalDays: dbTarget.SSLCriticalDays,
			SSLCheck:       dbTarget.SSLCheck,
			SSLGetChain:    dbTarget.SSLGetChain,
		}

		s.mu.Lock()
		s.targets[target.ID] = target
		s.mu.Unlock()

		go s.monitorTarget(target)
	}

	return nil
}

func (s *Service) GetStatus(targetID uint32) (*models.MonitorStatus, error) {
	db := database.GetDB()

	var status models.MonitorStatus
	if err := db.Where("target_id = ?", targetID).First(&status).Error; err != nil {
		return nil, err
	}

	return &status, nil
}

func (s *Service) ListStatus() []models.MonitorStatus {
	db := database.GetDB()

	var statuses []models.MonitorStatus
	db.Find(&statuses)

	return statuses
}