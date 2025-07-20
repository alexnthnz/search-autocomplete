package api

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupRouter configures and returns the HTTP router
func SetupRouter(handler *Handler, apiKey string, enableCORS bool) *gin.Engine {
	// Set Gin mode (release mode in production)
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery()) // Recover from panics

	if enableCORS {
		router.Use(handler.CORSMiddleware())
	}

	router.Use(handler.LoggingMiddleware())
	router.Use(handler.MetricsMiddleware())

	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Public endpoints
	v1 := router.Group("/api/v1")
	{
		// Autocomplete endpoints
		v1.GET("/autocomplete", handler.AutocompleteHandler)
		v1.POST("/autocomplete", handler.AutocompletePostHandler)

		// Health check
		v1.GET("/health", handler.HealthHandler)

		// Public stats (limited info)
		v1.GET("/stats", handler.StatsHandler)
	}

	// Admin endpoints (protected with API key if provided)
	admin := v1.Group("/admin")
	if apiKey != "" {
		admin.Use(handler.AuthMiddleware(apiKey))
	}
	{
		// Suggestion management
		admin.POST("/suggestions", handler.AddSuggestionHandler)
		admin.POST("/suggestions/batch", handler.BatchAddSuggestionsHandler)
		admin.PUT("/suggestions/:term/frequency", handler.UpdateFrequencyHandler)
		admin.DELETE("/suggestions/:term", handler.DeleteSuggestionHandler)
	}

	// Add a simple frontend for testing (optional)
	router.Static("/static", "./web/static")
	router.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})

	return router
}
