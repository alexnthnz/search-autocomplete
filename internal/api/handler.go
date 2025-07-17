package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"

	"github.com/alexnthnz/search-autocomplete/internal/service"
	"github.com/alexnthnz/search-autocomplete/pkg/models"
)

// Handler handles HTTP requests for the autocomplete API
type Handler struct {
	service     *service.AutocompleteService
	logger      *logrus.Logger
	rateLimiter *rate.Limiter
}

// NewHandler creates a new API handler
func NewHandler(service *service.AutocompleteService, logger *logrus.Logger) *Handler {
	// Rate limiter: 100 requests per second with burst of 200
	limiter := rate.NewLimiter(rate.Limit(100), 200)

	return &Handler{
		service:     service,
		logger:      logger,
		rateLimiter: limiter,
	}
}

// AutocompleteHandler handles autocomplete requests
func (h *Handler) AutocompleteHandler(c *gin.Context) {
	// Rate limiting
	if !h.rateLimiter.Allow() {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Rate limit exceeded",
		})
		return
	}

	// Parse query parameters
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Query parameter 'q' is required",
		})
		return
	}

	// Parse optional parameters
	limit := 10 // default
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	userID := c.Query("user_id")
	sessionID := c.Query("session_id")

	// Create request
	req := models.AutocompleteRequest{
		Query:     query,
		Limit:     limit,
		UserID:    userID,
		SessionID: sessionID,
	}

	// Get suggestions
	response, err := h.service.GetSuggestions(c.Request.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get suggestions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
		return
	}

	// Log the query for analytics
	go h.logQuery(query, userID, sessionID, c.ClientIP())

	c.JSON(http.StatusOK, response)
}

// AutocompletePostHandler handles POST requests for autocomplete
func (h *Handler) AutocompletePostHandler(c *gin.Context) {
	// Rate limiting
	if !h.rateLimiter.Allow() {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Rate limit exceeded",
		})
		return
	}

	var req models.AutocompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Set default limit
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 50 {
		req.Limit = 50
	}

	// Get suggestions
	response, err := h.service.GetSuggestions(c.Request.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get suggestions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
		return
	}

	// Log the query for analytics
	go h.logQuery(req.Query, req.UserID, req.SessionID, c.ClientIP())

	c.JSON(http.StatusOK, response)
}

// AddSuggestionHandler allows adding new suggestions (admin endpoint)
func (h *Handler) AddSuggestionHandler(c *gin.Context) {
	var suggestion models.Suggestion
	if err := c.ShouldBindJSON(&suggestion); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	if suggestion.Term == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Term is required",
		})
		return
	}

	// Set defaults
	if suggestion.UpdatedAt.IsZero() {
		suggestion.UpdatedAt = time.Now()
	}
	if suggestion.Score == 0 {
		suggestion.Score = float64(suggestion.Frequency)
	}

	if err := h.service.AddSuggestion(suggestion); err != nil {
		h.logger.WithError(err).Error("Failed to add suggestion")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to add suggestion",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Suggestion added successfully",
		"term":    suggestion.Term,
	})
}

// BatchAddSuggestionsHandler allows adding multiple suggestions at once
func (h *Handler) BatchAddSuggestionsHandler(c *gin.Context) {
	var suggestions []models.Suggestion
	if err := c.ShouldBindJSON(&suggestions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	if len(suggestions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No suggestions provided",
		})
		return
	}

	if len(suggestions) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Too many suggestions (max 1000)",
		})
		return
	}

	if err := h.service.BatchAddSuggestions(suggestions); err != nil {
		h.logger.WithError(err).Error("Failed to batch add suggestions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to add suggestions",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Suggestions added successfully",
		"count":   len(suggestions),
	})
}

// UpdateFrequencyHandler updates the frequency of a suggestion
func (h *Handler) UpdateFrequencyHandler(c *gin.Context) {
	term := c.Param("term")
	if term == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Term parameter is required",
		})
		return
	}

	frequencyStr := c.Query("frequency")
	if frequencyStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Frequency parameter is required",
		})
		return
	}

	frequency, err := strconv.ParseInt(frequencyStr, 10, 64)
	if err != nil || frequency < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid frequency value",
		})
		return
	}

	h.service.UpdateFrequency(term, frequency)

	c.JSON(http.StatusOK, gin.H{
		"message":   "Frequency updated successfully",
		"term":      term,
		"frequency": frequency,
	})
}

// DeleteSuggestionHandler removes a suggestion
func (h *Handler) DeleteSuggestionHandler(c *gin.Context) {
	term := c.Param("term")
	if term == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Term parameter is required",
		})
		return
	}

	deleted := h.service.DeleteSuggestion(term)
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Suggestion not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Suggestion deleted successfully",
		"term":    term,
	})
}

// StatsHandler returns service statistics
func (h *Handler) StatsHandler(c *gin.Context) {
	serviceStats := h.service.GetStats()
	trieStats := h.service.GetTrieStats()

	stats := gin.H{
		"service": serviceStats,
		"trie":    trieStats,
		"uptime":  time.Since(startTime).String(),
	}

	c.JSON(http.StatusOK, stats)
}

// HealthHandler provides health check endpoint
func (h *Handler) HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	})
}

// CORSMiddleware handles CORS headers
func (h *Handler) CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// LoggingMiddleware logs HTTP requests
func (h *Handler) LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		h.logger.WithFields(logrus.Fields{
			"status":     param.StatusCode,
			"method":     param.Method,
			"path":       param.Path,
			"ip":         param.ClientIP,
			"latency":    param.Latency,
			"user_agent": param.Request.UserAgent(),
		}).Info("HTTP Request")

		return ""
	})
}

// AuthMiddleware provides simple API key authentication for admin endpoints
func (h *Handler) AuthMiddleware(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if apiKey == "" {
			c.Next()
			return
		}

		providedKey := c.GetHeader("X-API-Key")
		if providedKey != apiKey {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or missing API key",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// logQuery logs search queries for analytics
func (h *Handler) logQuery(query, userID, sessionID, ipAddress string) {
	searchLog := models.SearchLog{
		Query:     query,
		UserID:    userID,
		SessionID: sessionID,
		Timestamp: time.Now(),
		IPAddress: ipAddress,
	}

	h.logger.WithFields(logrus.Fields{
		"query":      searchLog.Query,
		"user_id":    searchLog.UserID,
		"session_id": searchLog.SessionID,
		"ip":         searchLog.IPAddress,
	}).Info("Search query logged")

	// In a real implementation, you would save this to a database or message queue
	// for further processing and analytics
}

// startTime tracks when the server started
var startTime = time.Now()
