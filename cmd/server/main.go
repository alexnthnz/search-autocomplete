package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/alexnthnz/search-autocomplete/internal/api"
	"github.com/alexnthnz/search-autocomplete/internal/cache"
	"github.com/alexnthnz/search-autocomplete/internal/metrics"
	"github.com/alexnthnz/search-autocomplete/internal/pipeline"
	"github.com/alexnthnz/search-autocomplete/internal/service"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Set log level from environment
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		if parsedLevel, err := logrus.ParseLevel(level); err == nil {
			logger.SetLevel(parsedLevel)
		}
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	logger.Info("Starting Search Autocomplete Service")

	// Load configuration from environment variables
	config := loadConfig()

	// Create shared metrics instance first
	sharedMetrics := metrics.NewMetrics()

	// Initialize cache
	var cacheInstance cache.Cache
	if config.CacheEnabled {
		if config.RedisEnabled {
			redisConfig := cache.Config{
				Host:     config.RedisHost,
				Port:     config.RedisPort,
				Password: config.RedisPassword,
				DB:       config.RedisDB,
				TTL:      config.CacheTTL,
			}
			cacheInstance = cache.NewRedisCache(redisConfig, logger, sharedMetrics)
			logger.Info("Using Redis cache")
		} else {
			cacheInstance = cache.NewInMemoryCache(config.CacheTTL, logger, sharedMetrics)
			logger.Info("Using in-memory cache")
		}
	}

	// Initialize autocomplete service
	serviceConfig := service.Config{
		MaxSuggestions:  config.MaxSuggestions,
		EnableFuzzy:     config.EnableFuzzy,
		FuzzyThreshold:  config.FuzzyThreshold,
		CacheEnabled:    config.CacheEnabled,
		PersonalizedRec: config.PersonalizedRec,
	}

	autocompleteService := service.NewAutocompleteService(serviceConfig, cacheInstance, logger, sharedMetrics)

	// Load sample data
	autocompleteService.LoadSampleData()

	// Initialize data pipeline
	pipelineConfig := pipeline.Config{
		BatchSize:     config.PipelineBatchSize,
		FlushInterval: config.PipelineFlushInterval,
		QueueSize:     config.PipelineQueueSize,
	}

	dataPipeline := pipeline.NewDataPipeline(autocompleteService, pipelineConfig, logger, sharedMetrics)

	// Start data pipeline
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataPipeline.Start(ctx)
	defer dataPipeline.Stop()

	// Load historical data for testing
	go dataPipeline.LoadHistoricalData()

	// Initialize API handler and router
	apiHandler := api.NewHandler(autocompleteService, dataPipeline, logger, sharedMetrics)
	router := api.SetupRouter(apiHandler, config.APIKey, config.EnableCORS)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      router,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("port", config.Port).Info("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Print startup information
	printStartupInfo(config, logger)

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	}

	logger.Info("Server shutdown complete")
}

// Config holds application configuration
type Config struct {
	Port                  int
	APIKey                string
	EnableCORS            bool
	LogLevel              string
	ReadTimeout           time.Duration
	WriteTimeout          time.Duration
	IdleTimeout           time.Duration
	MaxSuggestions        int
	EnableFuzzy           bool
	FuzzyThreshold        int
	PersonalizedRec       bool
	CacheEnabled          bool
	CacheTTL              time.Duration
	RedisEnabled          bool
	RedisHost             string
	RedisPort             int
	RedisPassword         string
	RedisDB               int
	PipelineBatchSize     int
	PipelineFlushInterval time.Duration
	PipelineQueueSize     int
}

// loadConfig loads configuration from environment variables with defaults
func loadConfig() Config {
	config := Config{
		Port:                  8080,
		APIKey:                os.Getenv("API_KEY"),
		EnableCORS:            getEnvBool("ENABLE_CORS", true),
		LogLevel:              getEnvString("LOG_LEVEL", "info"),
		ReadTimeout:           getEnvDuration("READ_TIMEOUT", 10*time.Second),
		WriteTimeout:          getEnvDuration("WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:           getEnvDuration("IDLE_TIMEOUT", 60*time.Second),
		MaxSuggestions:        getEnvInt("MAX_SUGGESTIONS", 10),
		EnableFuzzy:           getEnvBool("ENABLE_FUZZY", true),
		FuzzyThreshold:        getEnvInt("FUZZY_THRESHOLD", 2),
		PersonalizedRec:       getEnvBool("PERSONALIZED_REC", false),
		CacheEnabled:          getEnvBool("CACHE_ENABLED", true),
		CacheTTL:              getEnvDuration("CACHE_TTL", 5*time.Minute),
		RedisEnabled:          getEnvBool("REDIS_ENABLED", false),
		RedisHost:             getEnvString("REDIS_HOST", "localhost"),
		RedisPort:             getEnvInt("REDIS_PORT", 6379),
		RedisPassword:         os.Getenv("REDIS_PASSWORD"),
		RedisDB:               getEnvInt("REDIS_DB", 0),
		PipelineBatchSize:     getEnvInt("PIPELINE_BATCH_SIZE", 100),
		PipelineFlushInterval: getEnvDuration("PIPELINE_FLUSH_INTERVAL", 30*time.Second),
		PipelineQueueSize:     getEnvInt("PIPELINE_QUEUE_SIZE", 10000),
	}

	// Override port if specified
	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Port = p
		}
	}

	return config
}

// Helper functions for environment variable parsing
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// printStartupInfo prints useful startup information
func printStartupInfo(config Config, logger *logrus.Logger) {
	logger.Info("🚀 Search Autocomplete Service Started Successfully!")
	logger.Info("==================================================")
	logger.Info("📋 Service Configuration:")
	logger.WithFields(logrus.Fields{
		"port":          config.Port,
		"cache_enabled": config.CacheEnabled,
		"redis_enabled": config.RedisEnabled,
		"fuzzy_enabled": config.EnableFuzzy,
		"cors_enabled":  config.EnableCORS,
		"api_key_set":   config.APIKey != "",
	}).Info("Configuration loaded")

	logger.Info("🔗 Available Endpoints:")
	logger.Info(fmt.Sprintf("  • Health Check:     GET  http://localhost:%d/api/v1/health", config.Port))
	logger.Info(fmt.Sprintf("  • Autocomplete:     GET  http://localhost:%d/api/v1/autocomplete?q=<query>", config.Port))
	logger.Info(fmt.Sprintf("  • Autocomplete:     POST http://localhost:%d/api/v1/autocomplete", config.Port))
	logger.Info(fmt.Sprintf("  • Statistics:       GET  http://localhost:%d/api/v1/stats", config.Port))
	logger.Info(fmt.Sprintf("  • Web Interface:    GET  http://localhost:%d/", config.Port))

	if config.APIKey != "" {
		logger.Info("🔒 Admin Endpoints (API Key Required):")
		logger.Info(fmt.Sprintf("  • Add Suggestion:   POST http://localhost:%d/api/v1/admin/suggestions", config.Port))
		logger.Info(fmt.Sprintf("  • Batch Add:        POST http://localhost:%d/api/v1/admin/suggestions/batch", config.Port))
		logger.Info(fmt.Sprintf("  • Update Frequency: PUT  http://localhost:%d/api/v1/admin/suggestions/<term>/frequency", config.Port))
		logger.Info(fmt.Sprintf("  • Delete:           DEL  http://localhost:%d/api/v1/admin/suggestions/<term>", config.Port))
	}

	logger.Info("==================================================")
	logger.Info("💡 Example Usage:")
	logger.Info(fmt.Sprintf("  curl 'http://localhost:%d/api/v1/autocomplete?q=app&limit=5'", config.Port))
	logger.Info("==================================================")
}
