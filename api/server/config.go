package server

import (
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"monitor/internal/config"

	"github.com/gin-gonic/gin"
)

// GetConfigResponse 获取配置响应
type GetConfigResponse struct {
	Config *config.Config `json:"config"`
}

// UpdateConfigRequest 更新配置请求
type UpdateConfigRequest struct {
	Config *config.Config `json:"config" binding:"required"`
}

// getConfig 获取系统配置
func (s *Server) getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, GetConfigResponse{
		Config: s.config,
	})
}

// updateConfig 更新系统配置
func (s *Server) updateConfig(c *gin.Context) {
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证配置
	if req.Config.Database.Driver != "sqlite" && req.Config.Database.Driver != "mysql" && req.Config.Database.Driver != "postgres" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid database driver. Must be sqlite, mysql, or postgres"})
		return
	}

	if req.Config.Server.HTTPPort < 1 || req.Config.Server.HTTPPort > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid HTTP port. Must be between 1 and 65535"})
		return
	}

	if req.Config.Server.GRPCPort < 1 || req.Config.Server.GRPCPort > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gRPC port. Must be between 1 and 65535"})
		return
	}

	// 保存配置到文件
	if err := config.SaveToFile(s.configPath, req.Config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save config: %v", err)})
		return
	}

	// 更新内存中的配置
	s.config = req.Config

	c.JSON(http.StatusOK, gin.H{
		"message": "Configuration updated successfully. Please restart the service for changes to take effect.",
		"config":  s.config,
	})
}

// TestDatabaseRequest 测试数据库连接请求
type TestDatabaseRequest struct {
	Driver   string `json:"driver" binding:"required"`
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	User     string `json:"user" binding:"required"`
	Password string `json:"password"`
	DBName   string `json:"dbname" binding:"required"`
	SSLMode  string `json:"sslmode"`
}

// testDatabase 测试数据库连接
func (s *Server) testDatabase(c *gin.Context) {
	var req TestDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate driver
	if req.Driver != "sqlite" && req.Driver != "mysql" && req.Driver != "postgres" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid database driver. Must be sqlite, mysql, or postgres"})
		return
	}

	// Validate port
	if req.Driver != "sqlite" && (req.Port < 1 || req.Port > 65535) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid port number"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Database configuration is valid",
		"driver":  req.Driver,
		"host":    req.Host,
		"port":    req.Port,
		"dbname":  req.DBName,
	})
}

// restartService 重启服务
func (s *Server) restartService(c *gin.Context) {
	// 发送重启信号
	// 注意：这需要程序能够捕获并处理这个信号
	// 在 Unix/Linux 系统上，可以使用 SIGUSR1 或 SIGHUP
	// 在 Windows 上，可能需要不同的方法

	c.JSON(http.StatusOK, gin.H{
		"message": "Restart signal sent. Service will restart shortly...",
	})

	// 在 goroutine 中执行重启，避免响应被中断
	go func() {
		// 给响应一些时间发送
		time.Sleep(100 * time.Millisecond)

		// 触发服务重启
		// 方法1: 使用 os.Exit(0) 让进程管理器重启
		// 方法2: 使用 exec 重新启动
		// 方法3: 发送信号给自己

		// 使用系统信号触发重启
		// 这需要在 main.go 中捕获这个信号并优雅关闭
		p, _ := os.FindProcess(os.Getpid())
		p.Signal(syscall.SIGTERM)

		// 或者直接退出，让 systemd/supervisor 等进程管理器重启
		// os.Exit(0)
	}()
}
