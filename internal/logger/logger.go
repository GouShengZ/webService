package logger

import (
	"io"
	"os"
	"path/filepath"

	"webservice/internal/config"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var log *logrus.Logger

// Init 初始化日志系统
func Init(cfg config.LogConfig) {
	log = logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// 设置日志格式
	if cfg.Format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	// 设置输出目标
	setupOutput(cfg)
}

// setupOutput 设置日志输出目标
func setupOutput(cfg config.LogConfig) {
	var writers []io.Writer

	// 控制台输出
	if cfg.Output == "console" || cfg.Output == "both" {
		writers = append(writers, os.Stdout)
	}

	// 文件输出
	if cfg.Output == "file" || cfg.Output == "both" {
		// 确保日志目录存在
		logDir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Errorf("Failed to create log directory: %v", err)
			return
		}

		// 配置日志轮转
		lumberjackLogger := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		writers = append(writers, lumberjackLogger)
	}

	// 设置多重输出
	if len(writers) > 0 {
		log.SetOutput(io.MultiWriter(writers...))
	}
}

// GetLogger 获取日志实例
func GetLogger() *logrus.Logger {
	return log
}

// Debug 调试级别日志
func Debug(args ...interface{}) {
	log.Debug(args...)
}

// Debugf 格式化调试级别日志
func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

// Info 信息级别日志
func Info(args ...interface{}) {
	log.Info(args...)
}

// Infof 格式化信息级别日志
func Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

// Warn 警告级别日志
func Warn(args ...interface{}) {
	log.Warn(args...)
}

// Warnf 格式化警告级别日志
func Warnf(format string, args ...interface{}) {
	log.Warnf(format, args...)
}

// Error 错误级别日志
func Error(args ...interface{}) {
	log.Error(args...)
}

// Errorf 格式化错误级别日志
func Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

// Fatal 致命错误级别日志
func Fatal(args ...interface{}) {
	log.Fatal(args...)
}

// Fatalf 格式化致命错误级别日志
func Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

// WithField 添加字段
func WithField(key string, value interface{}) *logrus.Entry {
	return log.WithField(key, value)
}

// WithFields 添加多个字段
func WithFields(fields logrus.Fields) *logrus.Entry {
	return log.WithFields(fields)
}
