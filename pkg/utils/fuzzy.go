package utils

import (
	"strings"
)

// FuzzyMatcher provides fuzzy string matching capabilities
type FuzzyMatcher struct {
	threshold int
}

// NewFuzzyMatcher creates a new fuzzy matcher with given threshold
func NewFuzzyMatcher(threshold int) *FuzzyMatcher {
	if threshold <= 0 {
		threshold = 2 // Default threshold
	}

	return &FuzzyMatcher{
		threshold: threshold,
	}
}

// LevenshteinDistance calculates the Levenshtein distance between two strings
func (f *FuzzyMatcher) LevenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create a 2D slice to store distances
	d := make([][]int, len(s1)+1)
	for i := range d {
		d[i] = make([]int, len(s2)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(s1); i++ {
		d[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		d[0][j] = j
	}

	// Fill the distance matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			d[i][j] = min(
				d[i-1][j]+1,      // deletion
				d[i][j-1]+1,      // insertion
				d[i-1][j-1]+cost, // substitution
			)
		}
	}

	return d[len(s1)][len(s2)]
}

// IsMatch checks if two strings match within the fuzzy threshold
func (f *FuzzyMatcher) IsMatch(s1, s2 string) bool {
	distance := f.LevenshteinDistance(strings.ToLower(s1), strings.ToLower(s2))
	return distance <= f.threshold
}

// GetSimilarity returns a similarity score between 0 and 1
func (f *FuzzyMatcher) GetSimilarity(s1, s2 string) float64 {
	distance := f.LevenshteinDistance(strings.ToLower(s1), strings.ToLower(s2))
	maxLen := max(len(s1), len(s2))

	if maxLen == 0 {
		return 1.0
	}

	return 1.0 - float64(distance)/float64(maxLen)
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// CommonPrefixLength returns the length of the common prefix between two strings
func CommonPrefixLength(s1, s2 string) int {
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	minLen := min(len(s1), len(s2), len(s1)) // Using existing min function
	for i := 0; i < minLen; i++ {
		if s1[i] != s2[i] {
			return i
		}
	}
	return minLen
}

// NormalizeQuery normalizes a search query for consistent processing
func NormalizeQuery(query string) string {
	// Convert to lowercase and trim whitespace
	query = strings.ToLower(strings.TrimSpace(query))

	// Remove extra spaces
	words := strings.Fields(query)
	return strings.Join(words, " ")
}

// IsSimilarEnough checks if a term is similar enough to the query based on various criteria
func IsSimilarEnough(query, term string, threshold float64) bool {
	if threshold <= 0 {
		threshold = 0.7 // Default threshold
	}

	query = NormalizeQuery(query)
	term = NormalizeQuery(term)

	// Exact match
	if query == term {
		return true
	}

	// Prefix match
	if strings.HasPrefix(term, query) {
		return true
	}

	// Fuzzy similarity
	matcher := NewFuzzyMatcher(2)
	similarity := matcher.GetSimilarity(query, term)

	return similarity >= threshold
}
