package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"monitor/api/server"
	"monitor/internal/config"
	"monitor/internal/database"
	"monitor/internal/elasticsearch"
	"monitor/internal/grpc"
	"monitor/internal/logger"
	"monitor/internal/monitor"

	"go.uber.org/zap"
)

var (
	configFile = flag.String("config", "etc/config.yaml", "Path to configuration file")
	version    = "1.0.0"
)

func main() {
	flag.Parse()

	// 加载配置
	var cfg *config.Config

	// 优先从配置文件加载，如果失败则从环境变量加载
	if _, err := os.Stat(*configFile); err == nil {
		cfg, err = config.LoadFromFile(*configFile)
		if err != nil {
			fmt.Printf("Failed to load config from file: %v\n", err)
			fmt.Println("Falling back to environment variables...")
			cfg = config.Load()
		}
	} else {
		fmt.Println("Config file not found, loading from environment variables...")
		cfg = config.Load()
	}

	// 初始化日志系统
	if err := logger.Init(cfg.Logger.Level, cfg.Logger.Output); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting Monitor Service",
		zap.String("version", version),
		zap.String("config_file", *configFile),
	)

	// 初始化数据库
	if err := database.InitDB(database.Config{
		Driver:   cfg.Database.Driver,
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}); err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	logger.Info("Database initialized",
		zap.String("driver", cfg.Database.Driver),
		zap.String("database", cfg.Database.DBName),
	)

	// 初始化 Elasticsearch（如果启用）
	var esClient *elasticsearch.Client
	if cfg.Elasticsearch.Enabled {
		var err error
		esClient, err = elasticsearch.NewClient(cfg.Elasticsearch)
		if err != nil {
			logger.Fatal("Failed to initialize Elasticsearch", zap.Error(err))
		}
		logger.Info("Elasticsearch initialized")
	} else {
		logger.Info("Elasticsearch is disabled")
	}

	// 创建索引模板（如果 ES 启用）
	if esClient != nil {
		if err := esClient.CreateIndexTemplate(); err != nil {
			logger.Warn("Failed to create index template", zap.Error(err))
		}
	}

	// 初始化监控服务
	monitorService := monitor.NewService(esClient)
	if err := monitorService.LoadTargetsFromDB(); err != nil {
		logger.Warn("Failed to load targets from database", zap.Error(err))
	} else {
		logger.Info("Monitor targets loaded")
	}

	// 创建等待组
	var wg sync.WaitGroup

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动HTTP服务器
	httpAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.HTTPPort)
	wg.Add(1)
	go func() {
		defer wg.Done()
		httpServer := server.NewServer(monitorService, esClient, *configFile, cfg)
		logger.Info("Starting HTTP server", zap.String("address", httpAddr))
		if err := httpServer.Run(httpAddr); err != nil {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// 启动gRPC服务器
	grpcAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.GRPCPort)
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting gRPC server", zap.String("address", grpcAddr))
		if err := grpc.StartServer(grpcAddr, monitorService); err != nil {
			logger.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	logger.Info("Monitor service is running",
		zap.Int("http_port", cfg.Server.HTTPPort),
		zap.Int("grpc_port", cfg.Server.GRPCPort),
	)

	// 等待信号
	sig := <-sigChan
	logger.Info("Received signal, shutting down...", zap.String("signal", sig.String()))

	// 优雅关闭
	logger.Info("Monitor service stopped")
}
