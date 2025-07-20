package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/alexnthnz/search-autocomplete/internal/cache"
	"github.com/alexnthnz/search-autocomplete/internal/metrics"
	"github.com/alexnthnz/search-autocomplete/internal/trie"
	"github.com/alexnthnz/search-autocomplete/pkg/models"
	"github.com/alexnthnz/search-autocomplete/pkg/utils"
)

// AutocompleteService provides autocomplete functionality
type AutocompleteService struct {
	trie         *trie.Trie
	cache        cache.Cache
	logger       *logrus.Logger
	fuzzyMatcher *utils.FuzzyMatcher
	metrics      *metrics.Metrics
}

// Config holds service configuration
type Config struct {
	MaxSuggestions  int
	EnableFuzzy     bool
	FuzzyThreshold  int
	CacheEnabled    bool
	PersonalizedRec bool
}

// NewAutocompleteService creates a new autocomplete service
func NewAutocompleteService(config Config, cache cache.Cache, logger *logrus.Logger, metrics *metrics.Metrics) *AutocompleteService {
	service := &AutocompleteService{
		trie:         trie.NewWithMetrics(metrics),
		cache:        cache,
		logger:       logger,
		fuzzyMatcher: utils.NewFuzzyMatcher(config.FuzzyThreshold),
		metrics:      metrics,
	}

	return service
}

// GetSuggestions returns autocomplete suggestions for a query
func (s *AutocompleteService) GetSuggestions(ctx context.Context, req models.AutocompleteRequest) (*models.AutocompleteResponse, error) {
	start := time.Now()
	defer func() {
		// Record request latency and count
		latency := time.Since(start)
		s.metrics.RecordRequest("autocomplete", "service", "200", latency)
	}()

	// Normalize query
	query := strings.ToLower(strings.TrimSpace(req.Query))
	if query == "" {
		return &models.AutocompleteResponse{
			Query:       req.Query,
			Suggestions: []models.Suggestion{},
			Latency:     time.Since(start).String(),
			Source:      "empty",
		}, nil
	}

	// Set default limit
	if req.Limit <= 0 {
		req.Limit = 10
	}

	var suggestions []models.Suggestion
	var source string

	// Try cache first
	if s.cache != nil {
		if cached, found := s.cache.Get(ctx, query); found {
			suggestions = cached
			source = "cache"
			s.logger.WithField("query", query).Debug("Cache hit")
		}
	}

	// If not in cache, search the trie
	if len(suggestions) == 0 {
		suggestions = s.trie.Search(query, req.Limit*2) // Get more for ranking
		source = "trie"
		s.logger.WithField("query", query).Debug("Trie search")

		// If no exact matches and fuzzy is enabled, try fuzzy matching
		if len(suggestions) == 0 && s.fuzzyMatcher != nil {
			suggestions = s.performFuzzySearch(query, req.Limit*2)
			if len(suggestions) > 0 {
				source = "fuzzy"
				s.metrics.RecordFuzzySearch()
				s.logger.WithField("query", query).Debug("Fuzzy search")
			}
		}

		// Cache the results
		if s.cache != nil && len(suggestions) > 0 {
			go func() {
				if err := s.cache.Set(context.Background(), query, suggestions); err != nil {
					s.logger.WithError(err).Error("Failed to cache suggestions")
					s.metrics.RecordError("service", "cache_set_failed")
				}
			}()
		}
	}

	// Apply personalization if enabled
	if req.UserID != "" || req.SessionID != "" {
		suggestions = s.personalizeResults(suggestions, req.UserID, req.SessionID)
	}

	// Apply ranking and limit
	suggestions = s.rankSuggestions(suggestions, query)
	if len(suggestions) > req.Limit {
		suggestions = suggestions[:req.Limit]
	}

	return &models.AutocompleteResponse{
		Query:       req.Query,
		Suggestions: suggestions,
		Latency:     time.Since(start).String(),
		Source:      source,
	}, nil
}

// AddSuggestion adds a new suggestion to the system
func (s *AutocompleteService) AddSuggestion(suggestion models.Suggestion) error {
	if suggestion.Term == "" {
		return nil
	}

	// Set default values
	if suggestion.UpdatedAt.IsZero() {
		suggestion.UpdatedAt = time.Now()
	}
	if suggestion.Score == 0 {
		suggestion.Score = float64(suggestion.Frequency)
	}

	s.trie.Insert(suggestion)
	s.logger.WithField("term", suggestion.Term).Debug("Added suggestion")

	return nil
}

// BatchAddSuggestions adds multiple suggestions efficiently
func (s *AutocompleteService) BatchAddSuggestions(suggestions []models.Suggestion) error {
	for _, suggestion := range suggestions {
		if err := s.AddSuggestion(suggestion); err != nil {
			s.logger.WithError(err).WithField("term", suggestion.Term).Error("Failed to add suggestion")
		}
	}
	return nil
}

// UpdateFrequency updates the frequency of a suggestion
func (s *AutocompleteService) UpdateFrequency(term string, frequency int64) {
	s.trie.UpdateFrequency(term, frequency)

	// Invalidate cache for all prefixes of this term
	if s.cache != nil {
		go s.invalidateCacheForTerm(term)
	}
}

// DeleteSuggestion removes a suggestion from the system
func (s *AutocompleteService) DeleteSuggestion(term string) bool {
	deleted := s.trie.Delete(term)

	if deleted && s.cache != nil {
		go s.invalidateCacheForTerm(term)
	}

	return deleted
}

// GetStats returns service statistics
func (s *AutocompleteService) GetStats() *metrics.Metrics {
	return s.metrics
}

// GetTrieStats returns trie-specific statistics
func (s *AutocompleteService) GetTrieStats() map[string]interface{} {
	return map[string]interface{}{
		"suggestions_count": s.trie.GetSuggestionsCount(),
	}
}

// performFuzzySearch performs fuzzy matching for queries with no exact matches
func (s *AutocompleteService) performFuzzySearch(query string, limit int) []models.Suggestion {
	// This is a simplified fuzzy search - in production, you'd want more sophisticated algorithms
	var fuzzyResults []models.Suggestion

	// Try removing last character (typo correction)
	if len(query) > 1 {
		shortened := query[:len(query)-1]
		results := s.trie.Search(shortened, limit)
		if len(results) > 0 {
			s.metrics.RecordFuzzyMatch()
		}
		fuzzyResults = append(fuzzyResults, results...)
	}

	// Try common substitutions
	commonSubs := map[string]string{
		"ph": "f", "f": "ph", "c": "k", "k": "c",
		"z": "s", "s": "z", "i": "y", "y": "i",
	}

	for old, new := range commonSubs {
		if strings.Contains(query, old) {
			modified := strings.ReplaceAll(query, old, new)
			results := s.trie.Search(modified, limit/2)
			if len(results) > 0 {
				s.metrics.RecordFuzzyMatch()
			}
			fuzzyResults = append(fuzzyResults, results...)
		}
	}

	// Reduce scores for fuzzy matches
	for i := range fuzzyResults {
		fuzzyResults[i].Score *= 0.8 // Penalty for fuzzy matches
	}

	return fuzzyResults
}

// personalizeResults applies personalization to suggestion results
func (s *AutocompleteService) personalizeResults(suggestions []models.Suggestion, userID, sessionID string) []models.Suggestion {
	// This is a simplified personalization - in production, you'd use ML models
	// or user behavior analysis

	if userID == "" && sessionID == "" {
		return suggestions
	}

	// Boost suggestions based on user context (placeholder implementation)
	for i := range suggestions {
		// Boost tech-related terms for tech users (example logic)
		if strings.Contains(suggestions[i].Category, "tech") {
			suggestions[i].Score *= 1.2
		}
	}

	return suggestions
}

// rankSuggestions applies advanced ranking to suggestions
func (s *AutocompleteService) rankSuggestions(suggestions []models.Suggestion, query string) []models.Suggestion {
	if len(suggestions) == 0 {
		return suggestions
	}

	// Calculate relevance scores
	for i := range suggestions {
		score := suggestions[i].Score

		// Boost exact prefix matches
		if strings.HasPrefix(strings.ToLower(suggestions[i].Term), query) {
			score *= 2.0
		}

		// Boost shorter terms (more likely to be what user wants)
		lengthBoost := 1.0 / (1.0 + float64(len(suggestions[i].Term))/10.0)
		score *= lengthBoost

		// Boost recent updates
		daysSinceUpdate := time.Since(suggestions[i].UpdatedAt).Hours() / 24
		if daysSinceUpdate < 7 {
			score *= 1.1
		}

		suggestions[i].Score = score
	}

	// Sort by final score
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Score > suggestions[j].Score
	})

	return suggestions
}

// invalidateCacheForTerm invalidates cache entries for all prefixes of a term
func (s *AutocompleteService) invalidateCacheForTerm(term string) {
	ctx := context.Background()
	term = strings.ToLower(term)

	// Invalidate all prefixes
	for i := 1; i <= len(term); i++ {
		prefix := term[:i]
		if err := s.cache.Delete(ctx, prefix); err != nil {
			s.logger.WithError(err).WithField("prefix", prefix).Error("Failed to invalidate cache")
		}
	}
}

// LoadSampleData loads sample suggestions for testing
func (s *AutocompleteService) LoadSampleData() {
	sampleSuggestions := []models.Suggestion{
		{Term: "apple", Frequency: 1000, Score: 1000, Category: "fruit", UpdatedAt: time.Now()},
		{Term: "application", Frequency: 800, Score: 800, Category: "tech", UpdatedAt: time.Now()},
		{Term: "app", Frequency: 1200, Score: 1200, Category: "tech", UpdatedAt: time.Now()},
		{Term: "amazon", Frequency: 900, Score: 900, Category: "company", UpdatedAt: time.Now()},
		{Term: "android", Frequency: 700, Score: 700, Category: "tech", UpdatedAt: time.Now()},
		{Term: "banana", Frequency: 600, Score: 600, Category: "fruit", UpdatedAt: time.Now()},
		{Term: "book", Frequency: 500, Score: 500, Category: "education", UpdatedAt: time.Now()},
		{Term: "basketball", Frequency: 400, Score: 400, Category: "sports", UpdatedAt: time.Now()},
		{Term: "computer", Frequency: 800, Score: 800, Category: "tech", UpdatedAt: time.Now()},
		{Term: "coding", Frequency: 600, Score: 600, Category: "tech", UpdatedAt: time.Now()},
	}

	s.BatchAddSuggestions(sampleSuggestions)
	s.logger.Info("Loaded sample data for autocomplete")
}
