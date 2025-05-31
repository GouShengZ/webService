package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"webservice/internal/config"
	"webservice/internal/database"
	"webservice/internal/logger"
	"webservice/internal/migration"
	"webservice/internal/minio"
	"webservice/internal/router"
	"webservice/internal/tracer"
)

// main 程序入口点
func main() {
	// 初始化配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	logger.Init(cfg.Log)
	logger.Info("Starting webservice...")

	// 初始化链路追踪
	closer, err := tracer.Init(cfg.Jaeger)
	if err != nil {
		logger.Warnf("Failed to initialize tracer (continuing without tracing): %v", err)
	} else {
		defer closer.Close()
		logger.Info("Tracer initialized successfully")
	}

	// 初始化数据库
	db, err := database.Init(cfg.Database)
	if err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	logger.Info("Database connected successfully")

	// 运行数据库迁移
	if err := migration.RunMigrations(db); err != nil {
		logger.Fatalf("Failed to run database migrations: %v", err)
	}
	logger.Info("Database migrations completed successfully")

	// 初始化MinIO客户端
	minioClient, err := minio.NewClient(cfg.MinIO)
	if err != nil {
		logger.Warnf("Failed to initialize MinIO client (continuing without file storage): %v", err)
		minioClient = nil // 设置为nil，让应用程序知道MinIO不可用
	} else {
		logger.Info("MinIO client initialized successfully")
	}

	// 初始化路由
	r := router.Setup(cfg, db, minioClient)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 启动服务器
	go func() {
		logger.Infof("Server starting on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// 优雅关闭，超时时间为30秒
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}
