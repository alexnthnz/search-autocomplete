package trie

import (
	"sort"
	"strings"
	"sync"

	"github.com/alexnthnz/search-autocomplete/internal/metrics"
	"github.com/alexnthnz/search-autocomplete/pkg/models"
)

// Trie represents the Trie data structure for autocomplete
type Trie struct {
	root    *models.TrieNode
	mutex   sync.RWMutex
	metrics *metrics.Metrics
	size    int // Track number of suggestions
}

// New creates a new Trie instance
func New() *Trie {
	return &Trie{
		root: &models.TrieNode{
			Children: make(map[rune]*models.TrieNode),
		},
		metrics: nil, // No metrics for backward compatibility
		size:    0,
	}
}

// NewWithMetrics creates a new Trie instance with provided metrics
func NewWithMetrics(metrics *metrics.Metrics) *Trie {
	return &Trie{
		root: &models.TrieNode{
			Children: make(map[rune]*models.TrieNode),
		},
		metrics: metrics,
		size:    0,
	}
}

// Insert adds a suggestion to the Trie
func (t *Trie) Insert(suggestion models.Suggestion) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	term := strings.ToLower(strings.TrimSpace(suggestion.Term))
	if term == "" {
		return
	}

	node := t.root
	for _, char := range term {
		if node.Children[char] == nil {
			node.Children[char] = &models.TrieNode{
				Children: make(map[rune]*models.TrieNode),
			}
		}
		node = node.Children[char]
		node.Frequency++
	}

	// Check if this is a new suggestion
	isNewSuggestion := !node.IsEndOfWord

	node.IsEndOfWord = true

	// Add or update suggestion in the node
	found := false
	for i := range node.Suggestions {
		if node.Suggestions[i].Term == suggestion.Term {
			node.Suggestions[i] = suggestion
			found = true
			break
		}
	}

	if !found {
		node.Suggestions = append(node.Suggestions, suggestion)
		if isNewSuggestion {
			t.size++
		}
	}

	// Sort suggestions by score (descending)
	sort.Slice(node.Suggestions, func(i, j int) bool {
		return node.Suggestions[i].Score > node.Suggestions[j].Score
	})

	// Record metrics
	if t.metrics != nil {
		t.metrics.RecordTrieInsert()
		t.metrics.UpdateTrieSize(t.size)
	}
}

// Search finds suggestions for a given prefix
func (t *Trie) Search(prefix string, limit int) []models.Suggestion {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if prefix == "" {
		return []models.Suggestion{}
	}

	// Navigate to the prefix node
	node := t.root
	for _, char := range prefix {
		if node.Children[char] == nil {
			// Record search with zero results
			if t.metrics != nil {
				t.metrics.RecordTrieSearch(0)
			}
			return []models.Suggestion{}
		}
		node = node.Children[char]
	}

	// Collect suggestions from this node and its descendants
	var suggestions []models.Suggestion
	t.collectSuggestions(node, prefix, &suggestions)

	// Sort by score (descending) and limit results
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Score > suggestions[j].Score
	})

	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	// Record search metrics with result count
	if t.metrics != nil {
		t.metrics.RecordTrieSearch(len(suggestions))
	}

	return suggestions
}

// collectSuggestions recursively collects all suggestions from a node and its descendants
func (t *Trie) collectSuggestions(node *models.TrieNode, currentWord string, suggestions *[]models.Suggestion) {
	if node.IsEndOfWord {
		*suggestions = append(*suggestions, node.Suggestions...)
	}

	for char, child := range node.Children {
		t.collectSuggestions(child, currentWord+string(char), suggestions)
	}
}

// GetSuggestionsCount returns the total number of unique suggestions in the trie
func (t *Trie) GetSuggestionsCount() int {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	count := 0
	t.countSuggestions(t.root, &count)
	return count
}

// countSuggestions recursively counts suggestions in the trie
func (t *Trie) countSuggestions(node *models.TrieNode, count *int) {
	if node.IsEndOfWord {
		*count += len(node.Suggestions)
	}

	for _, child := range node.Children {
		t.countSuggestions(child, count)
	}
}

// Delete removes a suggestion from the Trie
func (t *Trie) Delete(term string) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return false
	}

	deleted := t.deleteHelper(t.root, term, 0)

	if deleted {
		t.size--
		if t.metrics != nil {
			t.metrics.RecordTrieDelete()
			t.metrics.UpdateTrieSize(t.size)
		}
	}

	return deleted
}

// deleteHelper is a recursive helper for deletion
func (t *Trie) deleteHelper(node *models.TrieNode, term string, index int) bool {
	if index == len(term) {
		if !node.IsEndOfWord {
			return false
		}

		node.IsEndOfWord = false
		node.Suggestions = []models.Suggestion{}

		// If node has no children, it can be deleted
		return len(node.Children) == 0
	}

	char := rune(term[index])
	child, exists := node.Children[char]
	if !exists {
		return false
	}

	shouldDeleteChild := t.deleteHelper(child, term, index+1)

	if shouldDeleteChild {
		delete(node.Children, char)
		// Return true if current node has no children and is not end of another word
		return len(node.Children) == 0 && !node.IsEndOfWord
	}

	return false
}

// UpdateFrequency updates the frequency of a term in the trie
func (t *Trie) UpdateFrequency(term string, frequency int64) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return
	}

	node := t.root
	for _, char := range term {
		if node.Children[char] == nil {
			return // Term doesn't exist
		}
		node = node.Children[char]
	}

	if node.IsEndOfWord {
		for i := range node.Suggestions {
			if strings.ToLower(node.Suggestions[i].Term) == term {
				node.Suggestions[i].Frequency = frequency
				// Recalculate score based on frequency
				node.Suggestions[i].Score = float64(frequency) * 1.0 // Simple scoring
				break
			}
		}

		// Re-sort suggestions
		sort.Slice(node.Suggestions, func(i, j int) bool {
			return node.Suggestions[i].Score > node.Suggestions[j].Score
		})
	}
}
