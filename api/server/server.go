package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"monitor/api/middleware"
	"monitor/internal/alert"
	"monitor/internal/config"
	"monitor/internal/database"
	"monitor/internal/elasticsearch"
	"monitor/internal/logger"
	"monitor/internal/models"
	"monitor/internal/monitor"
	"monitor/pkg/ipgeo"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	router         *gin.Engine
	monitorService *monitor.Service
	ipgeoService   *ipgeo.Service
	es             *elasticsearch.Client
	alertService   *alert.Service
	configPath     string
	config         *config.Config
}

func NewServer(monitorService *monitor.Service, esClient *elasticsearch.Client, configPath string, cfg *config.Config) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Add timeout middleware
	router.Use(func(c *gin.Context) {
		// Set timeout for request processing (30 seconds)
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	server := &Server{
		router:         router,
		monitorService: monitorService,
		ipgeoService:   ipgeo.NewService(),
		es:             esClient,
		alertService:   alert.NewService(),
		configPath:     configPath,
		config:         cfg,
	}

	// Initialize file-based logging
	if err := logger.InitLogFileLog("logs"); err != nil {
		fmt.Printf("Warning: Failed to initialize file log: %v\n", err)
	}

	server.setupRoutes()

	return server
}

func (s *Server) setupRoutes() {
	// Apply rate limiting to all API routes
	api := s.router.Group("/api/v1")
	api.Use(middleware.RateLimit())

	{
		// Monitor management - all using POST
		api.POST("/monitor/add", s.addMonitor)
		api.POST("/monitor/list", s.listMonitors)
		api.POST("/monitor/get", s.getMonitor)
		api.POST("/monitor/update", s.updateMonitor)
		api.POST("/monitor/remove", s.removeMonitor)

		// Monitor status - using POST
		api.POST("/monitor/status/get", s.getMonitorStatus)
		api.POST("/monitor/status/list", s.listMonitorStatus)

		// Logs - using POST
		api.POST("/logs/search", s.searchLogs)
		api.POST("/logs/stats", s.getLogStats)

		// IP Geolocation - using POST and GET
		api.POST("/ipgeo/query", s.queryIPGeo)
		api.GET("/ip/geo/:ip", s.queryIPGeoGET)

		// DNS Providers - using POST
		api.POST("/dns/provider/add", s.addDNSProvider)
		api.POST("/dns/provider/list", s.listDNSProviders)
		api.POST("/dns/provider/get", s.getDNSProvider)
		api.POST("/dns/provider/update", s.updateDNSProvider)
		api.POST("/dns/provider/remove", s.removeDNSProvider)

		// Alert Channels - using POST
		api.POST("/alert/channel/add", s.addAlertChannel)
		api.POST("/alert/channel/list", s.listAlertChannels)
		api.POST("/alert/channel/get", s.getAlertChannel)
		api.POST("/alert/channel/update", s.updateAlertChannel)
		api.POST("/alert/channel/remove", s.removeAlertChannel)
		api.POST("/alert/channel/test", s.testAlertChannel)

		// Alert Rules - using POST
		api.POST("/alert/rule/add", s.addAlertRule)
		api.POST("/alert/rule/list", s.listAlertRules)
		api.POST("/alert/rule/get", s.getAlertRule)
		api.POST("/alert/rule/update", s.updateAlertRule)
		api.POST("/alert/rule/remove", s.removeAlertRule)
		api.POST("/alert/rule/listByTarget", s.listAlertRulesByTarget)

		// System Configuration
		api.GET("/config", s.getConfig)
		api.POST("/config", s.updateConfig)
		api.POST("/config/restart", s.restartService)
	}

	s.router.GET("/health", s.healthCheck)

	// Serve static files (no rate limiting for static content)
	s.router.Static("/static", "./web/static")

	// Load HTML template
	s.router.LoadHTMLFiles(
		"./web/templates/base.html",
		"./web/templates/index.html",
		"./web/templates/pages/dashboard.html",
		"./web/templates/pages/logs.html",
		"./web/templates/pages/alerts.html",
		"./web/templates/pages/settings.html",
	)

	// Frontend pages
	s.router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/dashboard")
	})
	s.router.GET("/dashboard", s.dashboardPage)
	s.router.GET("/logs", s.logsPage)
	s.router.GET("/alerts", s.alertsPage)
	s.router.GET("/settings", s.settingsPage)
}

// Common request/response types
type IDRequest struct {
	ID uint32 `json:"id" binding:"required"`
}

type AddMonitorRequest struct {
	Name     string            `json:"name" binding:"required"`
	Type     string            `json:"type" binding:"required,oneof=http https tcp udp dns ping smtp snmp ssl tls"`
	Address  string            `json:"address" binding:"required"`
	Port     int32             `json:"port"`
	Interval int64             `json:"interval"`
	Metadata map[string]string `json:"metadata"`
	Enabled  bool              `json:"enabled"`

	// HTTP/HTTPS specific fields
	HTTPMethod          string            `json:"http_method"`           // GET, POST, PUT, DELETE, etc.
	HTTPHeaders         map[string]string `json:"http_headers"`          // Custom headers
	HTTPBody            string            `json:"http_body"`             // Request body
	ResolvedHost        string            `json:"resolved_host"`         // Custom host resolution
	FollowRedirects     bool              `json:"follow_redirects"`      // Follow 301/302 redirects
	MaxRedirects        int               `json:"max_redirects"`         // Maximum redirect depth
	ExpectedStatusCodes string            `json:"expected_status_codes"` // Comma-separated status codes

	// DNS specific fields
	DNSServer     string `json:"dns_server"`      // Custom DNS server (e.g., 8.8.8.8:53)
	DNSServerName string `json:"dns_server_name"` // DNS server name (e.g., "Google DNS")
	DNSServerType string `json:"dns_server_type"` // DNS protocol: udp, tcp, doh, dot

	// PING specific fields
	PingCount   int `json:"ping_count"`   // Number of ping packets (default: 4)
	PingSize    int `json:"ping_size"`    // Size of ping packet in bytes (default: 32)
	PingTimeout int `json:"ping_timeout"` // Timeout in milliseconds (default: 5000)

	// SMTP specific fields
	SMTPUsername      string `json:"smtp_username"`       // SMTP authentication username
	SMTPPassword      string `json:"smtp_password"`       // SMTP authentication password
	SMTPUseTLS        bool   `json:"smtp_use_tls"`       // Use TLS/SSL (default: false)
	SMTPMailFrom      string `json:"smtp_mail_from"`     // From address for test
	SMTPMailTo        string `json:"smtp_mail_to"`       // To address for test
	SMTPCheckStartTLS bool   `json:"smtp_check_starttls"` // Check STARTTLS support (default: true)

	// SNMP specific fields
	SNMPCommunity    string `json:"snmp_community"`    // SNMP community string (default: public)
	SNMPOID          string `json:"snmp_oid"`           // SNMP OID to query
	SNMPVersion      string `json:"snmp_version"`        // SNMP version: v1, v2c, v3
	SNMPExpectedValue string `json:"snmp_expected_value"` // Expected value for comparison
	SNMPOperator     string `json:"snmp_operator"`       // eq, ne, gt, lt, ge, le

	// SSL/TLS specific fields
	SSLWarnDays    int  `json:"ssl_warn_days"`    // Days before expiration to warn (default: 30)
	SSLCriticalDays int  `json:"ssl_critical_days"`  // Days before expiration to mark as critical (default: 7)
	SSLCheck       bool `json:"ssl_check"`       // Enable SSL/TLS certificate monitoring
	SSLGetChain    bool `json:"ssl_get_chain"`   // Get certificate chain information
}

func (s *Server) addMonitor(c *gin.Context) {
	var req AddMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert request to database model
	target, err := ConvertAddRequestToModel(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert request"})
		return
	}

	if target.Interval == 0 {
		target.Interval = 60
	}

	db := database.GetDB()
	if err := db.Create(target).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create monitor"})
		return
	}

	// Convert model to monitor target
	monitorTarget, err := ConvertModelToMonitorTarget(*target)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert monitor target"})
		return
	}

	if err := s.monitorService.AddTarget(monitorTarget); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add monitor"})
		return
	}

	// Trigger immediate check after adding monitor
	go func() {
		time.Sleep(500 * time.Millisecond) // Small delay to ensure monitor is fully initialized
		if err := s.monitorService.TriggerCheck(monitorTarget.ID); err != nil {
			logger.Log.Warn("Failed to trigger initial check",
				zap.Uint32("target_id", monitorTarget.ID),
				zap.Error(err),
			)
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"id":      target.ID,
		"message": "Monitor created successfully",
	})
}

func (s *Server) listMonitors(c *gin.Context) {
	db := database.GetDB()

	var targets []models.MonitorTarget
	if err := db.Find(&targets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list monitors"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"targets": targets})
}

func (s *Server) getMonitor(c *gin.Context) {
	var req IDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()

	var target models.MonitorTarget
	if err := db.First(&target, req.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
		return
	}

	c.JSON(http.StatusOK, target)
}

func (s *Server) updateMonitor(c *gin.Context) {
	var req struct {
		IDRequest
		AddMonitorRequest
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()

	var target models.MonitorTarget
	if err := db.First(&target, req.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
		return
	}

	// Update model from request
	if err := UpdateModelFromRequest(&target, req.AddMonitorRequest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update monitor"})
		return
	}

	if err := db.Save(&target).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update monitor"})
		return
	}

	// Remove old target and add updated one
	if err := s.monitorService.RemoveTarget(target.ID); err == nil {
		monitorTarget, err := ConvertModelToMonitorTarget(target)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert monitor target"})
			return
		}
		s.monitorService.AddTarget(monitorTarget)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Monitor updated successfully"})
}

func (s *Server) removeMonitor(c *gin.Context) {
	var req IDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()

	// Start transaction
	tx := db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Delete related status records
	if err := tx.Where("target_id = ?", req.ID).Delete(&models.MonitorStatus{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete monitor status"})
		return
	}

	// Delete related history records
	if err := tx.Where("target_id = ?", req.ID).Delete(&models.MonitorHistory{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete monitor history"})
		return
	}

	// Delete the monitor target
	if err := tx.Delete(&models.MonitorTarget{}, req.ID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete monitor"})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Remove from monitoring service
	s.monitorService.RemoveTarget(req.ID)

	c.JSON(http.StatusOK, gin.H{"message": "Monitor deleted successfully"})
}

func (s *Server) getMonitorStatus(c *gin.Context) {
	var req IDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status, err := s.monitorService.GetStatus(req.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Status not found"})
		return
	}

	c.JSON(http.StatusOK, status)
}

func (s *Server) listMonitorStatus(c *gin.Context) {
	var req struct {
		TargetID *uint32 `json:"target_id,omitempty"`
		Limit    *int    `json:"limit,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// If binding fails, continue without filters (backward compatibility)
	}

	var statuses []*models.MonitorStatus
	db := database.GetDB()
	query := db.Order("checked_at DESC")

	if req.TargetID != nil {
		query = query.Where("target_id = ?", *req.TargetID)
	}

	if req.Limit != nil {
		query = query.Limit(*req.Limit)
	}

	if err := query.Find(&statuses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list monitor status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statuses": statuses})
}

type IPGeoRequest struct {
	IP string `json:"ip" binding:"required"`
}

func (s *Server) queryIPGeo(c *gin.Context) {
	var req IPGeoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.ipgeoService.QueryIP(req.IP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query IP geolocation"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (s *Server) queryIPGeoGET(c *gin.Context) {
	ip := c.Param("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "IP address is required"})
		return
	}

	result, err := s.ipgeoService.QueryIP(ip)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query IP geolocation"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// 日志查询相关的 API
type LogSearchRequest struct {
	TargetID   *uint32 `json:"target_id,omitempty"`
	Status     string  `json:"status,omitempty"`
	StartTime  *int64  `json:"start_time,omitempty"`  // Unix timestamp
	EndTime    *int64  `json:"end_time,omitempty"`    // Unix timestamp
	Size       int     `json:"size,omitempty"`
	From       int     `json:"from,omitempty"`
	QueryText  string  `json:"query_text,omitempty"`
}

func (s *Server) searchLogs(c *gin.Context) {
	var req LogSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// If ES is enabled, use ES; otherwise use file-based logs
	if s.es != nil {
		// 构建查询
		query := &elasticsearch.SearchQuery{
			TargetID:  req.TargetID,
			Status:    req.Status,
			Size:      req.Size,
			From:      req.From,
			QueryText: req.QueryText,
		}

		// 转换时间
		if req.StartTime != nil {
			t := time.Unix(*req.StartTime, 0)
			query.StartTime = &t
		}
		if req.EndTime != nil {
			t := time.Unix(*req.EndTime, 0)
			query.EndTime = &t
		}

		// 执行搜索
		result, err := s.es.SearchLogs(query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total": result.Total,
			"hits":  result.Hits,
		})
	} else {
		// Use file-based logs
		fileLogReq := &logger.LogQueryRequest{
			Status: req.Status,
			Limit:  req.Size,
			Offset: req.From,
		}

		// Convert TargetID from *uint32 to *int
		if req.TargetID != nil {
			id := int(*req.TargetID)
			fileLogReq.TargetID = &id
		}

		// 转换时间
		if req.StartTime != nil {
			t := time.Unix(*req.StartTime, 0)
			fileLogReq.StartTime = &t
		}
		if req.EndTime != nil {
			t := time.Unix(*req.EndTime, 0)
			fileLogReq.EndTime = &t
		}

		// Set default limit
		if fileLogReq.Limit <= 0 {
			fileLogReq.Limit = 100
		}

		// Query from file logs
		result, err := logger.QueryCheckLogs("logs", fileLogReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Convert file log entries to response format
		hits := make([]map[string]interface{}, 0)
		for _, entry := range result.Logs {
			source := map[string]interface{}{
				"target_id":     entry.TargetID,
				"target_name":   entry.TargetName,
				"target_type":   entry.Type,
				"address":       entry.Address,
				"status":        entry.Status,
				"response_time": entry.ResponseTime,
				"message":       entry.Message,
				"@timestamp":    entry.Timestamp.Format(time.RFC3339),
			}

			// Add request details if available
			if entry.Request != nil {
				source["request"] = entry.Request
			}

			// Add response details if available
			if entry.Response != nil {
				source["response"] = entry.Response
			}

			hit := map[string]interface{}{
				"_source": source,
			}
			hits = append(hits, hit)
		}

		c.JSON(http.StatusOK, gin.H{
			"total": result.Total,
			"hits":  hits,
		})
	}
}

type LogStatsRequest struct {
	TargetID  uint32 `json:"target_id" binding:"required"`
	StartTime int64  `json:"start_time"`  // Unix timestamp
	EndTime   int64  `json:"end_time"`    // Unix timestamp
}

func (s *Server) getLogStats(c *gin.Context) {
	if s.es == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Elasticsearch is not enabled"})
		return
	}

	var req LogStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 转换时间（默认最近24小时）
	startTime := time.Unix(req.StartTime, 0)
	if req.StartTime == 0 {
		startTime = time.Now().Add(-24 * time.Hour)
	}

	endTime := time.Unix(req.EndTime, 0)
	if req.EndTime == 0 {
		endTime = time.Now()
	}

	// 获取统计
	stats, err := s.es.GetLogStats(req.TargetID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

// DNS Provider management

type DNSProviderRequest struct {
	Name       string `json:"name" binding:"required"`       // Provider name
	Server     string `json:"server" binding:"required"`     // DNS server address
	ServerType string `json:"server_type" binding:"required"` // DNS protocol: udp, tcp, doh, dot
	IsDefault  bool   `json:"is_default"`                    // Mark as default
}

func (s *Server) addDNSProvider(c *gin.Context) {
	var req DNSProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()

	// If setting as default, unset other defaults
	if req.IsDefault {
		db.Model(&models.DNSProvider{}).Where("is_default = ?", true).Update("is_default", false)
	}

	provider := models.DNSProvider{
		Name:       req.Name,
		Server:     req.Server,
		ServerType: req.ServerType,
		IsDefault:  req.IsDefault,
	}

	if err := db.Create(&provider).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create DNS provider"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      provider.ID,
		"message": "DNS provider created successfully",
	})
}

func (s *Server) listDNSProviders(c *gin.Context) {
	db := database.GetDB()

	var providers []models.DNSProvider
	if err := db.Find(&providers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list DNS providers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

func (s *Server) getDNSProvider(c *gin.Context) {
	var req IDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()

	var provider models.DNSProvider
	if err := db.First(&provider, req.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "DNS provider not found"})
		return
	}

	c.JSON(http.StatusOK, provider)
}

func (s *Server) updateDNSProvider(c *gin.Context) {
	var req struct {
		IDRequest
		DNSProviderRequest
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()

	var provider models.DNSProvider
	if err := db.First(&provider, req.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "DNS provider not found"})
		return
	}

	// If setting as default, unset other defaults
	if req.IsDefault {
		db.Model(&models.DNSProvider{}).Where("is_default = ? AND id != ?", true, req.ID).Update("is_default", false)
	}

	provider.Name = req.Name
	provider.Server = req.Server
	provider.ServerType = req.ServerType
	provider.IsDefault = req.IsDefault

	if err := db.Save(&provider).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update DNS provider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "DNS provider updated successfully"})
}

func (s *Server) removeDNSProvider(c *gin.Context) {
	var req IDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()

	if err := db.Delete(&models.DNSProvider{}, req.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete DNS provider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "DNS provider deleted successfully"})
}
// Alert Channel API handlers

func (s *Server) addAlertChannel(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		Type    string `json:"type" binding:"required"`
		Enabled bool   `json:"enabled"`
		Config  string `json:"config" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	channel := models.AlertChannel{
		Name:    req.Name,
		Type:    req.Type,
		Enabled: req.Enabled,
		Config:  req.Config,
	}

	db := database.GetDB()
	if err := db.Create(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create alert channel"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": channel.ID, "message": "Alert channel created successfully"})
}

func (s *Server) listAlertChannels(c *gin.Context) {
	db := database.GetDB()
	var channels []models.AlertChannel
	if err := db.Find(&channels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list alert channels"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"channels": channels})
}

func (s *Server) getAlertChannel(c *gin.Context) {
	var req IDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	var channel models.AlertChannel
	if err := db.First(&channel, req.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert channel not found"})
		return
	}
	c.JSON(http.StatusOK, channel)
}

func (s *Server) updateAlertChannel(c *gin.Context) {
	var req struct {
		IDRequest
		Name    string `json:"name" binding:"required"`
		Type    string `json:"type" binding:"required"`
		Enabled bool   `json:"enabled"`
		Config  string `json:"config" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	var channel models.AlertChannel
	if err := db.First(&channel, req.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert channel not found"})
		return
	}

	channel.Name = req.Name
	channel.Type = req.Type
	channel.Enabled = req.Enabled
	channel.Config = req.Config

	if err := db.Save(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update alert channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alert channel updated successfully"})
}

func (s *Server) removeAlertChannel(c *gin.Context) {
	var req IDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	if err := db.Delete(&models.AlertChannel{}, req.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete alert channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alert channel deleted successfully"})
}

func (s *Server) testAlertChannel(c *gin.Context) {
	var req IDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.alertService.TestAlertChannel(uint(req.ID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send test alert"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Test alert sent successfully"})
}

// Alert Rule API handlers

func (s *Server) addAlertRule(c *gin.Context) {
	var req struct {
		TargetID       uint32 `json:"target_id" binding:"required"`
		ChannelID      uint   `json:"channel_id" binding:"required"`
		ThresholdType  string `json:"threshold_type" binding:"required"`
		ThresholdValue int    `json:"threshold_value" binding:"required"`
		Enabled        bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule := models.AlertRule{
		TargetID:       req.TargetID,
		ChannelID:      req.ChannelID,
		ThresholdType:  req.ThresholdType,
		ThresholdValue: req.ThresholdValue,
		Enabled:        req.Enabled,
	}

	db := database.GetDB()
	if err := db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create alert rule"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": rule.ID, "message": "Alert rule created successfully"})
}

func (s *Server) listAlertRules(c *gin.Context) {
	db := database.GetDB()
	var rules []models.AlertRule
	if err := db.Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list alert rules"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

func (s *Server) getAlertRule(c *gin.Context) {
	var req IDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	var rule models.AlertRule
	if err := db.First(&rule, req.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert rule not found"})
		return
	}
	c.JSON(http.StatusOK, rule)
}

func (s *Server) updateAlertRule(c *gin.Context) {
	var req struct {
		IDRequest
		TargetID       uint32 `json:"target_id" binding:"required"`
		ChannelID      uint   `json:"channel_id" binding:"required"`
		ThresholdType  string `json:"threshold_type" binding:"required"`
		ThresholdValue int    `json:"threshold_value" binding:"required"`
		Enabled        bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	var rule models.AlertRule
	if err := db.First(&rule, req.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert rule not found"})
		return
	}

	rule.TargetID = req.TargetID
	rule.ChannelID = req.ChannelID
	rule.ThresholdType = req.ThresholdType
	rule.ThresholdValue = req.ThresholdValue
	rule.Enabled = req.Enabled

	if err := db.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update alert rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alert rule updated successfully"})
}

func (s *Server) removeAlertRule(c *gin.Context) {
	var req IDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	if err := db.Delete(&models.AlertRule{}, req.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete alert rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alert rule deleted successfully"})
}

func (s *Server) listAlertRulesByTarget(c *gin.Context) {
	var req struct {
		TargetID uint32 `json:"target_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rules, err := s.alertService.ListAlertRulesByTarget(req.TargetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list alert rules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// Frontend page handlers
func (s *Server) dashboardPage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"Title":    "监控列表 - Monitor Dashboard",
		"ActiveTab": "monitors",
	})
}

func (s *Server) logsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"Title":    "日志查询 - Monitor Dashboard",
		"ActiveTab": "logs",
	})
}

func (s *Server) alertsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"Title":    "告警管理 - Monitor Dashboard",
		"ActiveTab": "alerts",
	})
}

func (s *Server) settingsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"Title":    "系统设置 - Monitor Dashboard",
		"ActiveTab": "settings",
	})
}
