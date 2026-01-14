package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"monitor/internal/config"
	"monitor/internal/logger"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type LogEntry struct {
	TargetID     uint32                 `json:"target_id"`
	TargetName   string                 `json:"target_name"`
	TargetType   string                 `json:"target_type"`
	Address      string                 `json:"address"`
	Status       string                 `json:"status"` // up, down, unknown
	ResponseTime int64                  `json:"response_time"` // milliseconds
	Message      string                 `json:"message"`
	Timestamp    time.Time              `json:"@timestamp"`

	// 请求信息
	Request struct {
		Method      string            `json:"method,omitempty"`
		Headers     map[string]string `json:"headers,omitempty"`
		Body        string            `json:"body,omitempty"`
		ResolvedURL string            `json:"resolved_url,omitempty"`
	} `json:"request"`

	// 响应信息
	Response struct {
		StatusCode    int               `json:"status_code,omitempty"`
		Headers       map[string]string `json:"headers,omitempty"`
		Body          string            `json:"body,omitempty"`
		ContentLength int64             `json:"content_length,omitempty"`
	} `json:"response"`

	// 错误信息
	Error struct {
		Type    string `json:"type,omitempty"`
		Message string `json:"message,omitempty"`
	} `json:"error,omitempty"`

	// 额外元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Client struct {
	es         *elasticsearch.Client
	config     config.ElasticsearchConfig
	indexName  string
}

func NewClient(cfg config.ElasticsearchConfig) (*Client, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	esConfig := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
	}

	es, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	// 测试连接
	res, err := es.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch returned error: %s", res.String())
	}

	// 生成索引名称（按日期滚动）
	indexName := fmt.Sprintf("%s-%s", cfg.IndexPrefix, time.Now().Format("2006.01.02"))

	client := &Client{
		es:        es,
		config:    cfg,
		indexName: indexName,
	}

	logger.Log.Info("Elasticsearch client initialized successfully")
	logger.Log.Debug(fmt.Sprintf("ES addresses: %v", cfg.Addresses))

	return client, nil
}

// IndexLog 索引日志到 Elasticsearch
func (c *Client) IndexLog(entry *LogEntry) error {
	if c == nil || c.es == nil {
		return nil // ES 未启用，跳过
	}

	// 更新索引名称（支持按日期滚动）
	c.indexName = fmt.Sprintf("%s-%s", c.config.IndexPrefix, time.Now().Format("2006.01.02"))

	// 设置时间戳
	entry.Timestamp = time.Now().UTC()

	// 序列化为 JSON
	body, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	// 索引文档
	req := esapi.IndexRequest{
		Index:      c.indexName,
		DocumentID: "", // 让 ES 自动生成 ID
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}

	res, err := req.Do(context.Background(), c.es)
	if err != nil {
		return fmt.Errorf("failed to index log: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch indexing error: %s", res.String())
	}

	logger.Log.Debug(fmt.Sprintf("Log indexed to ES: index=%s, target_id=%d, status=%s",
		c.indexName, entry.TargetID, entry.Status))

	return nil
}

// SearchLogs 搜索日志
type SearchQuery struct {
	TargetID   *uint32    `json:"target_id,omitempty"`
	Status     string     `json:"status,omitempty"`
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	Size       int        `json:"size,omitempty"`
	From       int        `json:"from,omitempty"`
	QueryText  string     `json:"query_text,omitempty"`
}

type SearchResult struct {
	Total int64       `json:"total"`
	Hits  []LogEntry  `json:"hits"`
}

func (c *Client) SearchLogs(query *SearchQuery) (*SearchResult, error) {
	if c == nil || c.es == nil {
		return &SearchResult{Total: 0, Hits: []LogEntry{}}, nil
	}

	// 构建查询
	boolQuery := map[string]interface{}{
		"bool": map[string]interface{}{
			"must": []map[string]interface{}{},
		},
	}

	mustQueries := boolQuery["bool"].(map[string]interface{})["must"].([]map[string]interface{})

	// 目标 ID 过滤
	if query.TargetID != nil {
		mustQueries = append(mustQueries, map[string]interface{}{
			"term": map[string]interface{}{
				"target_id": *query.TargetID,
			},
		})
	}

	// 状态过滤
	if query.Status != "" {
		mustQueries = append(mustQueries, map[string]interface{}{
			"term": map[string]interface{}{
				"status": query.Status,
			},
		})
	}

	// 时间范围过滤
	if query.StartTime != nil || query.EndTime != nil {
		rangeQuery := map[string]interface{}{}
		if query.StartTime != nil {
			rangeQuery["gte"] = query.StartTime.Format(time.RFC3339)
		}
		if query.EndTime != nil {
			rangeQuery["lte"] = query.EndTime.Format(time.RFC3339)
		}
		mustQueries = append(mustQueries, map[string]interface{}{
			"range": map[string]interface{}{
				"@timestamp": rangeQuery,
			},
		})
	}

	// 全文搜索
	if query.QueryText != "" {
		mustQueries = append(mustQueries, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query": query.QueryText,
				"fields": []string{"message", "request.body", "response.body", "error.message"},
			},
		})
	}

	boolQuery["bool"].(map[string]interface{})["must"] = mustQueries

	// 设置分页
	if query.Size <= 0 {
		query.Size = 20
	}
	if query.Size > 100 {
		query.Size = 100 // 最大 100 条
	}

	searchBody := map[string]interface{}{
		"query": boolQuery,
		"size":  query.Size,
		"from":  query.From,
		"sort": []map[string]interface{}{
			{"@timestamp": map[string]interface{}{"order": "desc"}},
		},
	}

	body, err := json.Marshal(searchBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search query: %w", err)
	}

	// 执行搜索（使用索引通配符）
	indexPattern := fmt.Sprintf("%s-*", c.config.IndexPrefix)
	req := esapi.SearchRequest{
		Index: []string{indexPattern},
		Body:  bytes.NewReader(body),
	}

	res, err := req.Do(context.Background(), c.es)
	if err != nil {
		return nil, fmt.Errorf("failed to search logs: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch search error: %s", res.String())
	}

	// 解析响应
	var response struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source LogEntry `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	// 提取结果
	result := &SearchResult{
		Total: response.Hits.Total.Value,
		Hits:  make([]LogEntry, 0, len(response.Hits.Hits)),
	}

	for _, hit := range response.Hits.Hits {
		result.Hits = append(result.Hits, hit.Source)
	}

	logger.Log.Debug(fmt.Sprintf("Log search completed: total=%d, returned=%d",
		result.Total, len(result.Hits)))

	return result, nil
}

// GetLogStats 获取日志统计信息
func (c *Client) GetLogStats(targetID uint32, startTime, endTime time.Time) (map[string]interface{}, error) {
	if c == nil || c.es == nil {
		return map[string]interface{}{}, nil
	}

	// 构建聚合查询
	query := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"target_id": targetID,
						},
					},
					{
						"range": map[string]interface{}{
							"@timestamp": map[string]interface{}{
								"gte": startTime.Format(time.RFC3339),
								"lte": endTime.Format(time.RFC3339),
							},
						},
					},
				},
			},
		},
		"aggs": map[string]interface{}{
			"status_count": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "status",
				},
			},
			"avg_response_time": map[string]interface{}{
				"avg": map[string]interface{}{
					"field": "response_time",
				},
			},
		},
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stats query: %w", err)
	}

	indexPattern := fmt.Sprintf("%s-*", c.config.IndexPrefix)
	req := esapi.SearchRequest{
		Index: []string{indexPattern},
		Body:  bytes.NewReader(body),
	}

	res, err := req.Do(context.Background(), c.es)
	if err != nil {
		return nil, fmt.Errorf("failed to get log stats: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch stats error: %s", res.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to parse stats response: %w", err)
	}

	return response, nil
}

// CreateIndexTemplate 创建索引模板（如果不存在）
func (c *Client) CreateIndexTemplate() error {
	if c == nil || c.es == nil {
		return nil
	}

	templateName := fmt.Sprintf("%s-template", c.config.IndexPrefix)

	template := map[string]interface{}{
		"index_patterns": []string{fmt.Sprintf("%s-*", c.config.IndexPrefix)},
		"template": map[string]interface{}{
			"settings": map[string]interface{}{
				"number_of_shards":   1,
				"number_of_replicas": 1,
				"refresh_interval":   "5s",
			},
			"mappings": map[string]interface{}{
				"properties": map[string]interface{}{
					"target_id":      map[string]string{"type": "integer"},
					"target_name":    map[string]string{"type": "keyword"},
					"target_type":    map[string]string{"type": "keyword"},
					"address":        map[string]string{"type": "keyword"},
					"status":         map[string]string{"type": "keyword"},
					"response_time":  map[string]string{"type": "long"},
					"message":        map[string]string{"type": "text"},
					"@timestamp":     map[string]string{"type": "date"},
					"request":        map[string]string{"type": "object"},
					"response":       map[string]string{"type": "object"},
					"error":          map[string]string{"type": "object"},
					"metadata":       map[string]string{"type": "object"},
				},
			},
		},
	}

	body, err := json.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal index template: %w", err)
	}

	req := esapi.IndicesPutIndexTemplateRequest{
		Name: templateName,
		Body: bytes.NewReader(body),
	}

	res, err := req.Do(context.Background(), c.es)
	if err != nil {
		return fmt.Errorf("failed to create index template: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 404 { // 404 表示已存在
		logger.Log.Warn(fmt.Sprintf("Failed to create index template: %s", res.String()))
	} else {
		logger.Log.Info(fmt.Sprintf("Index template created: %s", templateName))
	}

	return nil
}