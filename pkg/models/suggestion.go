package models

import "time"

// Suggestion represents an autocomplete suggestion
type Suggestion struct {
	Term      string    `json:"term"`
	Frequency int64     `json:"frequency"`
	Score     float64   `json:"score"`
	Category  string    `json:"category,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AutocompleteRequest represents a request for autocomplete suggestions
type AutocompleteRequest struct {
	Query     string `json:"query" binding:"required"`
	Limit     int    `json:"limit,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// AutocompleteResponse represents the response containing suggestions
type AutocompleteResponse struct {
	Query       string       `json:"query"`
	Suggestions []Suggestion `json:"suggestions"`
	Latency     string       `json:"latency"`
	Source      string       `json:"source"` // "cache" or "index"
}

// SearchLog represents a search query log entry
type SearchLog struct {
	Query     string    `json:"query"`
	UserID    string    `json:"user_id,omitempty"`
	SessionID string    `json:"session_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	IPAddress string    `json:"ip_address,omitempty"`
}

// TrieNode represents a node in the Trie structure
type TrieNode struct {
	Children    map[rune]*TrieNode `json:"children"`
	IsEndOfWord bool               `json:"is_end_of_word"`
	Suggestions []Suggestion       `json:"suggestions"`
	Frequency   int64              `json:"frequency"`
}
