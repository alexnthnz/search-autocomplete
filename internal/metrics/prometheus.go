package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricsInstance *Metrics
	metricsOnce     sync.Once
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// Request metrics
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	ActiveRequests  prometheus.Gauge

	// Cache metrics
	CacheHitsTotal   *prometheus.CounterVec
	CacheMissesTotal *prometheus.CounterVec
	CacheOperations  *prometheus.HistogramVec

	// Trie metrics
	TrieSearches *prometheus.CounterVec
	TrieInserts  prometheus.Counter
	TrieDeletes  prometheus.Counter
	TrieSize     prometheus.Gauge

	// Fuzzy search metrics
	FuzzySearches prometheus.Counter
	FuzzyMatches  prometheus.Counter

	// Pipeline metrics
	PipelineProcessed *prometheus.CounterVec
	PipelineQueueSize prometheus.Gauge
	PipelineLatency   *prometheus.HistogramVec

	// Error metrics
	ErrorsTotal *prometheus.CounterVec
}

// NewMetrics creates a new metrics instance (singleton)
func NewMetrics() *Metrics {
	metricsOnce.Do(func() {
		metricsInstance = &Metrics{
			// Request metrics
			RequestsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "autocomplete_requests_total",
					Help: "Total number of autocomplete requests",
				},
				[]string{"method", "endpoint", "status"},
			),
			RequestDuration: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "autocomplete_request_duration_seconds",
					Help:    "Duration of autocomplete requests",
					Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
				},
				[]string{"method", "endpoint"},
			),
			ActiveRequests: promauto.NewGauge(
				prometheus.GaugeOpts{
					Name: "autocomplete_active_requests",
					Help: "Number of active requests",
				},
			),

			// Cache metrics
			CacheHitsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "autocomplete_cache_hits_total",
					Help: "Total number of cache hits",
				},
				[]string{"cache_type"},
			),
			CacheMissesTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "autocomplete_cache_misses_total",
					Help: "Total number of cache misses",
				},
				[]string{"cache_type"},
			),
			CacheOperations: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "autocomplete_cache_operation_duration_seconds",
					Help:    "Duration of cache operations",
					Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
				},
				[]string{"operation", "cache_type"},
			),

			// Trie metrics
			TrieSearches: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "autocomplete_trie_searches_total",
					Help: "Total number of trie searches",
				},
				[]string{"result_count"},
			),
			TrieInserts: promauto.NewCounter(
				prometheus.CounterOpts{
					Name: "autocomplete_trie_inserts_total",
					Help: "Total number of trie insertions",
				},
			),
			TrieDeletes: promauto.NewCounter(
				prometheus.CounterOpts{
					Name: "autocomplete_trie_deletes_total",
					Help: "Total number of trie deletions",
				},
			),
			TrieSize: promauto.NewGauge(
				prometheus.GaugeOpts{
					Name: "autocomplete_trie_size",
					Help: "Current size of the trie",
				},
			),

			// Fuzzy search metrics
			FuzzySearches: promauto.NewCounter(
				prometheus.CounterOpts{
					Name: "autocomplete_fuzzy_searches_total",
					Help: "Total number of fuzzy searches performed",
				},
			),
			FuzzyMatches: promauto.NewCounter(
				prometheus.CounterOpts{
					Name: "autocomplete_fuzzy_matches_total",
					Help: "Total number of fuzzy matches found",
				},
			),

			// Pipeline metrics
			PipelineProcessed: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "autocomplete_pipeline_processed_total",
					Help: "Total number of items processed by pipeline",
				},
				[]string{"stage"},
			),
			PipelineQueueSize: promauto.NewGauge(
				prometheus.GaugeOpts{
					Name: "autocomplete_pipeline_queue_size",
					Help: "Current size of the pipeline queue",
				},
			),
			PipelineLatency: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "autocomplete_pipeline_latency_seconds",
					Help:    "Pipeline processing latency",
					Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
				},
				[]string{"stage"},
			),

			// Error metrics
			ErrorsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "autocomplete_errors_total",
					Help: "Total number of errors",
				},
				[]string{"component", "error_type"},
			),
		}
	})
	return metricsInstance
}

// RecordRequest records a request metric
func (m *Metrics) RecordRequest(method, endpoint, status string, duration time.Duration) {
	m.RequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	m.RequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// IncActiveRequests increments active requests
func (m *Metrics) IncActiveRequests() {
	m.ActiveRequests.Inc()
}

// DecActiveRequests decrements active requests
func (m *Metrics) DecActiveRequests() {
	m.ActiveRequests.Dec()
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit(cacheType string) {
	m.CacheHitsTotal.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss(cacheType string) {
	m.CacheMissesTotal.WithLabelValues(cacheType).Inc()
}

// RecordCacheOperation records a cache operation duration
func (m *Metrics) RecordCacheOperation(operation, cacheType string, duration time.Duration) {
	m.CacheOperations.WithLabelValues(operation, cacheType).Observe(duration.Seconds())
}

// RecordTrieSearch records a trie search
func (m *Metrics) RecordTrieSearch(resultCount int) {
	var label string
	switch {
	case resultCount == 0:
		label = "zero"
	case resultCount <= 5:
		label = "few"
	case resultCount <= 20:
		label = "many"
	default:
		label = "lots"
	}
	m.TrieSearches.WithLabelValues(label).Inc()
}

// RecordTrieInsert records a trie insertion
func (m *Metrics) RecordTrieInsert() {
	m.TrieInserts.Inc()
}

// RecordTrieDelete records a trie deletion
func (m *Metrics) RecordTrieDelete() {
	m.TrieDeletes.Inc()
}

// UpdateTrieSize updates the trie size gauge
func (m *Metrics) UpdateTrieSize(size int) {
	m.TrieSize.Set(float64(size))
}

// RecordFuzzySearch records a fuzzy search
func (m *Metrics) RecordFuzzySearch() {
	m.FuzzySearches.Inc()
}

// RecordFuzzyMatch records a fuzzy match
func (m *Metrics) RecordFuzzyMatch() {
	m.FuzzyMatches.Inc()
}

// RecordPipelineProcessed records pipeline processing
func (m *Metrics) RecordPipelineProcessed(stage string) {
	m.PipelineProcessed.WithLabelValues(stage).Inc()
}

// UpdatePipelineQueueSize updates pipeline queue size
func (m *Metrics) UpdatePipelineQueueSize(size int) {
	m.PipelineQueueSize.Set(float64(size))
}

// RecordPipelineLatency records pipeline latency
func (m *Metrics) RecordPipelineLatency(stage string, duration time.Duration) {
	m.PipelineLatency.WithLabelValues(stage).Observe(duration.Seconds())
}

// RecordError records an error
func (m *Metrics) RecordError(errorType, component string) {
	m.ErrorsTotal.WithLabelValues(errorType, component).Inc()
}
