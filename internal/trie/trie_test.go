package trie

import (
	"testing"
	"time"

	"github.com/alexnthnz/search-autocomplete/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestTrie_Insert_And_Search(t *testing.T) {
	trie := New()

	// Test data
	suggestions := []models.Suggestion{
		{
			Term:      "apple",
			Frequency: 100,
			Score:     100,
			Category:  "fruit",
			UpdatedAt: time.Now(),
		},
		{
			Term:      "application",
			Frequency: 200,
			Score:     200,
			Category:  "tech",
			UpdatedAt: time.Now(),
		},
		{
			Term:      "app",
			Frequency: 300,
			Score:     300,
			Category:  "tech",
			UpdatedAt: time.Now(),
		},
	}

	// Insert suggestions
	for _, suggestion := range suggestions {
		trie.Insert(suggestion)
	}

	// Test search with "app" prefix
	results := trie.Search("app", 10)
	assert.True(t, len(results) >= 2, "Should find at least 2 suggestions for 'app'")

	// Verify results are sorted by score (descending)
	for i := 1; i < len(results); i++ {
		assert.True(t, results[i-1].Score >= results[i].Score, "Results should be sorted by score")
	}

	// Test exact match
	exactResults := trie.Search("apple", 10)
	assert.True(t, len(exactResults) >= 1, "Should find exact match for 'apple'")
	assert.Equal(t, "apple", exactResults[0].Term, "First result should be exact match")

	// Test no results
	noResults := trie.Search("xyz", 10)
	assert.Equal(t, 0, len(noResults), "Should find no results for 'xyz'")

	// Test empty query
	emptyResults := trie.Search("", 10)
	assert.Equal(t, 0, len(emptyResults), "Should find no results for empty query")
}

func TestTrie_Delete(t *testing.T) {
	trie := New()

	suggestion := models.Suggestion{
		Term:      "test",
		Frequency: 100,
		Score:     100,
		UpdatedAt: time.Now(),
	}

	// Insert and verify
	trie.Insert(suggestion)
	results := trie.Search("test", 10)
	assert.Equal(t, 1, len(results), "Should find the inserted suggestion")

	// Delete and verify
	deleted := trie.Delete("test")
	assert.True(t, deleted, "Delete should return true for existing term")

	results = trie.Search("test", 10)
	assert.Equal(t, 0, len(results), "Should find no results after deletion")

	// Try to delete non-existent term
	deleted = trie.Delete("nonexistent")
	assert.False(t, deleted, "Delete should return false for non-existent term")
}

func TestTrie_UpdateFrequency(t *testing.T) {
	trie := New()

	suggestion := models.Suggestion{
		Term:      "test",
		Frequency: 100,
		Score:     100,
		UpdatedAt: time.Now(),
	}

	trie.Insert(suggestion)

	// Update frequency
	trie.UpdateFrequency("test", 500)

	// Verify updated frequency
	results := trie.Search("test", 10)
	assert.Equal(t, 1, len(results), "Should find the suggestion")
	assert.Equal(t, int64(500), results[0].Frequency, "Frequency should be updated")
	assert.Equal(t, float64(500), results[0].Score, "Score should be updated based on frequency")
}

func TestTrie_GetSuggestionsCount(t *testing.T) {
	trie := New()

	// Initially should be 0
	count := trie.GetSuggestionsCount()
	assert.Equal(t, 0, count, "Initial count should be 0")

	// Add suggestions
	suggestions := []models.Suggestion{
		{Term: "apple", Frequency: 100, Score: 100, UpdatedAt: time.Now()},
		{Term: "application", Frequency: 200, Score: 200, UpdatedAt: time.Now()},
		{Term: "app", Frequency: 300, Score: 300, UpdatedAt: time.Now()},
	}

	for _, suggestion := range suggestions {
		trie.Insert(suggestion)
	}

	// Should have 3 suggestions
	count = trie.GetSuggestionsCount()
	assert.Equal(t, 3, count, "Should have 3 suggestions after insertion")
}

func TestTrie_CaseInsensitive(t *testing.T) {
	trie := New()

	suggestion := models.Suggestion{
		Term:      "Apple",
		Frequency: 100,
		Score:     100,
		UpdatedAt: time.Now(),
	}

	trie.Insert(suggestion)

	// Search with different cases
	testCases := []string{"apple", "APPLE", "Apple", "aPpLe"}

	for _, testCase := range testCases {
		results := trie.Search(testCase, 10)
		assert.True(t, len(results) >= 1, "Should find results for case: %s", testCase)
	}
}

func BenchmarkTrie_Insert(b *testing.B) {
	trie := New()
	suggestion := models.Suggestion{
		Term:      "benchmark",
		Frequency: 100,
		Score:     100,
		UpdatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		suggestion.Term = "benchmark" + string(rune(i))
		trie.Insert(suggestion)
	}
}

func BenchmarkTrie_Search(b *testing.B) {
	trie := New()

	// Pre-populate with test data
	for i := 0; i < 1000; i++ {
		suggestion := models.Suggestion{
			Term:      "test" + string(rune(i)),
			Frequency: int64(i),
			Score:     float64(i),
			UpdatedAt: time.Now(),
		}
		trie.Insert(suggestion)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Search("test", 10)
	}
}
