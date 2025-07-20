package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"

	"github.com/alexnthnz/search-autocomplete/internal/metrics"
	"github.com/alexnthnz/search-autocomplete/internal/pipeline"
	"github.com/alexnthnz/search-autocomplete/internal/service"
	"github.com/alexnthnz/search-autocomplete/pkg/errors"
	"github.com/alexnthnz/search-autocomplete/pkg/models"
	"github.com/alexnthnz/search-autocomplete/pkg/utils"
)

var startTime time.Time

func init() {
	startTime = time.Now()
}

// Handler handles HTTP requests for the autocomplete API
type Handler struct {
	service     *service.AutocompleteService
	logger      *logrus.Logger
	rateLimiter *rate.Limiter
	validator   *utils.QueryValidator
	metrics     *metrics.Metrics
	pipeline    *pipeline.DataPipeline
}

// NewHandler creates a new API handler
func NewHandler(service *service.AutocompleteService, pipeline *pipeline.DataPipeline, logger *logrus.Logger, metricsInstance *metrics.Metrics) *Handler {
	// Rate limiter: 100 requests per second with burst of 200
	limiter := rate.NewLimiter(rate.Limit(100), 200)

	return &Handler{
		service:     service,
		logger:      logger,
		rateLimiter: limiter,
		validator:   utils.NewQueryValidator(),
		metrics:     metricsInstance,
		pipeline:    pipeline,
	}
}

// AutocompleteHandler handles autocomplete requests
func (h *Handler) AutocompleteHandler(c *gin.Context) {
	// Rate limiting
	if !h.rateLimiter.Allow() {
		apiErr := errors.NewRateLimitError()
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	// Parse query parameters
	query := c.Query("q")
	if query == "" {
		apiErr := errors.NewValidationError("Query parameter 'q' is required", "Missing required parameter")
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	// Validate and sanitize query
	if err := h.validator.ValidateQuery(query); err != nil {
		apiErr := errors.NewValidationError("Invalid query", err.Error())
		h.metrics.RecordError("api", "validation_failed")
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	// Sanitize the query
	query = h.validator.SanitizeQuery(query)

	// Parse optional parameters
	limit := 10 // default
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	userID := c.Query("user_id")
	sessionID := c.Query("session_id")

	// Validate userID and sessionID if provided
	if userID != "" {
		if err := utils.ValidateUserID(userID); err != nil {
			apiErr := errors.NewValidationError("Invalid user ID", err.Error())
			c.JSON(apiErr.HTTPStatus, apiErr)
			return
		}
	}

	if sessionID != "" {
		if err := utils.ValidateSessionID(sessionID); err != nil {
			apiErr := errors.NewValidationError("Invalid session ID", err.Error())
			c.JSON(apiErr.HTTPStatus, apiErr)
			return
		}
	}

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
		h.metrics.RecordError("api", "service_failed")
		apiErr := errors.NewInternalError("Failed to process request", err)
		c.JSON(apiErr.HTTPStatus, apiErr)
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
		apiErr := errors.NewRateLimitError()
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	var req models.AutocompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiErr := errors.NewValidationError("Invalid request body", err.Error())
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	// Validate and sanitize query
	if err := h.validator.ValidateQuery(req.Query); err != nil {
		apiErr := errors.NewValidationError("Invalid query", err.Error())
		h.metrics.RecordError("api", "validation_failed")
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	// Sanitize the query
	req.Query = h.validator.SanitizeQuery(req.Query)

	// Validate userID and sessionID if provided
	if req.UserID != "" {
		if err := utils.ValidateUserID(req.UserID); err != nil {
			apiErr := errors.NewValidationError("Invalid user ID", err.Error())
			c.JSON(apiErr.HTTPStatus, apiErr)
			return
		}
	}

	if req.SessionID != "" {
		if err := utils.ValidateSessionID(req.SessionID); err != nil {
			apiErr := errors.NewValidationError("Invalid session ID", err.Error())
			c.JSON(apiErr.HTTPStatus, apiErr)
			return
		}
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
		h.metrics.RecordError("api", "service_failed")
		apiErr := errors.NewInternalError("Failed to process request", err)
		c.JSON(apiErr.HTTPStatus, apiErr)
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
		apiErr := errors.NewValidationError("Invalid request body", err.Error())
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	if suggestion.Term == "" {
		apiErr := errors.NewValidationError("Term is required", "Suggestion term cannot be empty")
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	// Validate the term
	if err := utils.ValidateTerm(suggestion.Term); err != nil {
		apiErr := errors.NewValidationError("Invalid term", err.Error())
		c.JSON(apiErr.HTTPStatus, apiErr)
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
		h.metrics.RecordError("api", "service_failed")
		apiErr := errors.NewInternalError("Failed to add suggestion", err)
		c.JSON(apiErr.HTTPStatus, apiErr)
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
		apiErr := errors.NewValidationError("Invalid request body", err.Error())
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	if len(suggestions) == 0 {
		apiErr := errors.NewValidationError("No suggestions provided", "Request body must contain at least one suggestion")
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	if len(suggestions) > 1000 {
		apiErr := errors.NewValidationError("Too many suggestions", "Maximum 1000 suggestions allowed per batch")
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	// Validate all terms
	for i, suggestion := range suggestions {
		if err := utils.ValidateTerm(suggestion.Term); err != nil {
			apiErr := errors.NewValidationError("Invalid term in batch", fmt.Sprintf("Suggestion %d: %s", i+1, err.Error()))
			c.JSON(apiErr.HTTPStatus, apiErr)
			return
		}
	}

	if err := h.service.BatchAddSuggestions(suggestions); err != nil {
		h.logger.WithError(err).Error("Failed to batch add suggestions")
		h.metrics.RecordError("api", "service_failed")
		apiErr := errors.NewInternalError("Failed to add suggestions", err)
		c.JSON(apiErr.HTTPStatus, apiErr)
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
		apiErr := errors.NewValidationError("Term parameter is required", "URL path must include term parameter")
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	// Validate the term
	if err := utils.ValidateTerm(term); err != nil {
		apiErr := errors.NewValidationError("Invalid term", err.Error())
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	frequencyStr := c.Query("frequency")
	if frequencyStr == "" {
		apiErr := errors.NewValidationError("Frequency parameter is required", "Query parameter 'frequency' is required")
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	frequency, err := strconv.ParseInt(frequencyStr, 10, 64)
	if err != nil || frequency < 0 {
		apiErr := errors.NewValidationError("Invalid frequency value", "Frequency must be a non-negative integer")
		c.JSON(apiErr.HTTPStatus, apiErr)
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
		apiErr := errors.NewValidationError("Term parameter is required", "URL path must include term parameter")
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	// Validate the term
	if err := utils.ValidateTerm(term); err != nil {
		apiErr := errors.NewValidationError("Invalid term", err.Error())
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	deleted := h.service.DeleteSuggestion(term)
	if !deleted {
		apiErr := errors.NewNotFoundError("suggestion")
		c.JSON(apiErr.HTTPStatus, apiErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Suggestion deleted successfully",
		"term":    term,
	})
}

// StatsHandler returns service statistics
func (h *Handler) StatsHandler(c *gin.Context) {
	// Get Prometheus metrics and convert to compatible format
	serviceStats := gin.H{
		"TotalQueries":      0, // We could extract from Prometheus metrics if needed
		"CacheHits":         0, // We could extract from Prometheus metrics if needed
		"CacheMisses":       0, // We could extract from Prometheus metrics if needed
		"ActiveRequests":    h.metrics.ActiveRequests,
		"RequestsTotal":     h.metrics.RequestsTotal,
		"RequestDuration":   h.metrics.RequestDuration,
		"CacheHitsTotal":    h.metrics.CacheHitsTotal,
		"CacheMissesTotal":  h.metrics.CacheMissesTotal,
		"CacheOperations":   h.metrics.CacheOperations,
		"TrieSearches":      h.metrics.TrieSearches,
		"TrieInserts":       h.metrics.TrieInserts,
		"TrieDeletes":       h.metrics.TrieDeletes,
		"TrieSize":          h.metrics.TrieSize,
		"FuzzySearches":     h.metrics.FuzzySearches,
		"FuzzyMatches":      h.metrics.FuzzyMatches,
		"PipelineProcessed": h.metrics.PipelineProcessed,
		"PipelineQueueSize": h.metrics.PipelineQueueSize,
		"PipelineLatency":   h.metrics.PipelineLatency,
		"ErrorsTotal":       h.metrics.ErrorsTotal,
	}

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

// MetricsMiddleware records request metrics
func (h *Handler) MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		h.metrics.IncActiveRequests()

		c.Next()

		duration := time.Since(start)
		status := strconv.Itoa(c.Writer.Status())

		h.metrics.RecordRequest(c.Request.Method, c.FullPath(), status, duration)
		h.metrics.DecActiveRequests()
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

	// Send to data pipeline for processing
	if h.pipeline != nil {
		if err := h.pipeline.LogQuery(searchLog); err != nil {
			h.logger.WithError(err).Warn("Failed to send log to pipeline")
		}
	}
}
