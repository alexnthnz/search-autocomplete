package pipeline

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/alexnthnz/search-autocomplete/internal/metrics"
	"github.com/alexnthnz/search-autocomplete/internal/service"
	"github.com/alexnthnz/search-autocomplete/pkg/models"
)

// DataPipeline processes search logs and updates suggestions
type DataPipeline struct {
	service       *service.AutocompleteService
	logger        *logrus.Logger
	logQueue      chan models.SearchLog
	freqUpdates   map[string]int64
	freqMutex     sync.RWMutex
	batchSize     int
	flushInterval time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
	metrics       *metrics.Metrics
}

// Config holds pipeline configuration
type Config struct {
	BatchSize     int
	FlushInterval time.Duration
	QueueSize     int
}

// NewDataPipeline creates a new data processing pipeline
func NewDataPipeline(service *service.AutocompleteService, config Config, logger *logrus.Logger, metricsInstance *metrics.Metrics) *DataPipeline {
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = 30 * time.Second
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 10000
	}

	return &DataPipeline{
		service:       service,
		logger:        logger,
		logQueue:      make(chan models.SearchLog, config.QueueSize),
		freqUpdates:   make(map[string]int64),
		batchSize:     config.BatchSize,
		flushInterval: config.FlushInterval,
		stopChan:      make(chan struct{}),
		metrics:       metricsInstance,
	}
}

// Start begins processing search logs
func (p *DataPipeline) Start(ctx context.Context) {
	p.logger.Info("Starting data pipeline")

	// Start log processor
	p.wg.Add(1)
	go p.processLogs(ctx)

	// Start frequency updater
	p.wg.Add(1)
	go p.updateFrequencies(ctx)

	// Start trending detector
	p.wg.Add(1)
	go p.detectTrending(ctx)
}

// Stop gracefully shuts down the pipeline
func (p *DataPipeline) Stop() {
	p.logger.Info("Stopping data pipeline")
	close(p.stopChan)
	p.wg.Wait()
	p.logger.Info("Data pipeline stopped")
}

// LogQuery adds a search query to the processing queue
func (p *DataPipeline) LogQuery(log models.SearchLog) error {
	select {
	case p.logQueue <- log:
		// Update queue size metric
		p.metrics.UpdatePipelineQueueSize(len(p.logQueue))
		return nil
	default:
		p.logger.Warn("Log queue is full, dropping log")
		p.metrics.RecordError("pipeline", "queue_full")
		return fmt.Errorf("log queue is full")
	}
}

// processLogs processes incoming search logs
func (p *DataPipeline) processLogs(ctx context.Context) {
	defer p.wg.Done()

	logs := make([]models.SearchLog, 0, p.batchSize)
	ticker := time.NewTicker(p.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.processBatch(logs)
			return
		case <-p.stopChan:
			p.processBatch(logs)
			return
		case log := <-p.logQueue:
			logs = append(logs, log)
			if len(logs) >= p.batchSize {
				p.processBatch(logs)
				logs = logs[:0] // Clear slice
			}
		case <-ticker.C:
			if len(logs) > 0 {
				p.processBatch(logs)
				logs = logs[:0]
			}
		}
	}
}

// processBatch processes a batch of search logs
func (p *DataPipeline) processBatch(logs []models.SearchLog) {
	if len(logs) == 0 {
		return
	}

	start := time.Now()
	p.logger.WithField("count", len(logs)).Debug("Processing log batch")

	queryFreq := make(map[string]int64)

	// Aggregate query frequencies
	for _, log := range logs {
		query := normalizeQuery(log.Query)
		if query != "" {
			queryFreq[query]++
		}
	}

	// Update frequency tracking
	p.freqMutex.Lock()
	for query, count := range queryFreq {
		p.freqUpdates[query] += count
	}
	p.freqMutex.Unlock()

	// Extract and add new suggestions from queries
	p.extractNewSuggestions(queryFreq)

	// Record processing metrics
	p.metrics.RecordPipelineProcessed("batch")
	p.metrics.RecordPipelineLatency("batch", time.Since(start))
	p.metrics.UpdatePipelineQueueSize(len(p.logQueue))
}

// updateFrequencies periodically updates suggestion frequencies
func (p *DataPipeline) updateFrequencies(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.flushInterval * 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.flushFrequencyUpdates()
			return
		case <-p.stopChan:
			p.flushFrequencyUpdates()
			return
		case <-ticker.C:
			p.flushFrequencyUpdates()
		}
	}
}

// flushFrequencyUpdates applies accumulated frequency updates
func (p *DataPipeline) flushFrequencyUpdates() {
	start := time.Now()

	p.freqMutex.Lock()
	updates := make(map[string]int64)
	for query, count := range p.freqUpdates {
		updates[query] = count
	}
	p.freqUpdates = make(map[string]int64) // Clear the map
	p.freqMutex.Unlock()

	if len(updates) == 0 {
		return
	}

	p.logger.WithField("count", len(updates)).Debug("Flushing frequency updates")

	for query, count := range updates {
		p.service.UpdateFrequency(query, count)
	}

	// Record flush metrics
	p.metrics.RecordPipelineProcessed("frequency_flush")
	p.metrics.RecordPipelineLatency("frequency_flush", time.Since(start))
}

// extractNewSuggestions identifies potential new suggestions from search queries
func (p *DataPipeline) extractNewSuggestions(queryFreq map[string]int64) {
	for query, freq := range queryFreq {
		// Skip very short or very long queries
		if len(query) < 2 || len(query) > 50 {
			continue
		}

		// Create suggestion with basic scoring
		suggestion := models.Suggestion{
			Term:      query,
			Frequency: freq,
			Score:     float64(freq),
			Category:  p.categorizeQuery(query),
			UpdatedAt: time.Now(),
		}

		// Add as potential suggestion
		p.service.AddSuggestion(suggestion)
	}
}

// detectTrending identifies trending search terms
func (p *DataPipeline) detectTrending(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(time.Hour) // Check for trends hourly
	defer ticker.Stop()

	recentQueries := make(map[string][]time.Time)
	var mutex sync.RWMutex

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.analyzeTrends(recentQueries, &mutex)
		}
	}
}

// analyzeTrends analyzes query patterns to identify trending terms
func (p *DataPipeline) analyzeTrends(recentQueries map[string][]time.Time, mutex *sync.RWMutex) {
	mutex.Lock()
	defer mutex.Unlock()

	now := time.Now()
	hourAgo := now.Add(-time.Hour)
	dayAgo := now.Add(-24 * time.Hour)

	trending := make(map[string]float64)

	for query, timestamps := range recentQueries {
		// Clean old timestamps
		var recent []time.Time
		for _, ts := range timestamps {
			if ts.After(dayAgo) {
				recent = append(recent, ts)
			}
		}
		recentQueries[query] = recent

		if len(recent) < 5 { // Need minimum queries to consider trending
			continue
		}

		// Count queries in last hour vs last day
		hourCount := 0
		dayCount := len(recent)

		for _, ts := range recent {
			if ts.After(hourAgo) {
				hourCount++
			}
		}

		// Calculate trend score (recent activity vs historical)
		if dayCount > hourCount {
			trendScore := float64(hourCount) / float64(dayCount-hourCount)
			if trendScore > 1.5 { // Trending threshold
				trending[query] = trendScore
			}
		}
	}

	// Boost trending terms
	for query, score := range trending {
		currentFreq := int64(len(recentQueries[query]))
		boostedFreq := int64(float64(currentFreq) * (1.0 + score))
		p.service.UpdateFrequency(query, boostedFreq)

		p.logger.WithFields(logrus.Fields{
			"query":       query,
			"trend_score": score,
			"frequency":   boostedFreq,
		}).Info("Detected trending query")
	}
}

// categorizeQuery attempts to categorize a search query
func (p *DataPipeline) categorizeQuery(query string) string {
	query = strings.ToLower(query)

	// Simple keyword-based categorization
	techTerms := []string{"app", "software", "computer", "tech", "programming", "code", "api", "web", "mobile", "android", "ios"}
	for _, term := range techTerms {
		if strings.Contains(query, term) {
			return "tech"
		}
	}

	businessTerms := []string{"company", "business", "service", "product", "market", "sales", "marketing"}
	for _, term := range businessTerms {
		if strings.Contains(query, term) {
			return "business"
		}
	}

	entertainmentTerms := []string{"movie", "music", "game", "video", "show", "entertainment", "sport", "book"}
	for _, term := range entertainmentTerms {
		if strings.Contains(query, term) {
			return "entertainment"
		}
	}

	return "general"
}

// normalizeQuery normalizes search queries for consistent processing
func normalizeQuery(query string) string {
	// Convert to lowercase and trim
	query = strings.ToLower(strings.TrimSpace(query))

	// Remove special characters and extra spaces
	words := strings.FieldsFunc(query, func(c rune) bool {
		return !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == ' ')
	})

	return strings.Join(words, " ")
}

// GetStats returns pipeline statistics
func (p *DataPipeline) GetStats() map[string]interface{} {
	p.freqMutex.RLock()
	pendingUpdates := len(p.freqUpdates)
	p.freqMutex.RUnlock()

	return map[string]interface{}{
		"queue_length":    len(p.logQueue),
		"pending_updates": pendingUpdates,
		"batch_size":      p.batchSize,
		"flush_interval":  p.flushInterval.String(),
	}
}

// LoadHistoricalData simulates loading historical search data
func (p *DataPipeline) LoadHistoricalData() {
	// Simulate historical search queries for testing
	historicalQueries := []string{
		"apple", "application", "app", "android", "amazon",
		"banana", "book", "basketball", "computer", "coding",
		"developer", "design", "database", "facebook", "google",
		"iphone", "javascript", "java", "machine learning", "mobile",
		"netflix", "python", "programming", "react", "software",
		"technology", "web development", "youtube", "zoom",
	}

	baseTime := time.Now().Add(-30 * 24 * time.Hour) // 30 days ago

	for i, query := range historicalQueries {
		for j := 0; j < (i%10+1)*100; j++ { // Varying frequencies
			log := models.SearchLog{
				Query:     query,
				UserID:    fmt.Sprintf("user_%d", j%1000),
				SessionID: fmt.Sprintf("session_%d", j%500),
				Timestamp: baseTime.Add(time.Duration(j) * time.Hour),
				IPAddress: fmt.Sprintf("192.168.1.%d", j%255),
			}

			// Don't block if queue is full during historical load
			select {
			case p.logQueue <- log:
			default:
				// Skip if queue is full
			}
		}
	}

	p.logger.WithField("queries", len(historicalQueries)).Info("Loaded historical search data")
}
