package utils

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

var (
	ErrInvalidQuery      = errors.New("invalid query: contains forbidden characters")
	ErrQueryTooLong      = errors.New("query too long")
	ErrQueryTooShort     = errors.New("query too short")
	ErrInvalidCharacters = errors.New("query contains invalid characters")
	ErrSuspiciousPattern = errors.New("query contains suspicious patterns")
)

const (
	MaxQueryLength = 100
	MinQueryLength = 1
)

// QueryValidator validates search queries for security
type QueryValidator struct {
	maxLength       int
	minLength       int
	allowedPattern  *regexp.Regexp
	blockedPatterns []*regexp.Regexp
}

// NewQueryValidator creates a new query validator
func NewQueryValidator() *QueryValidator {
	// Allow alphanumeric, spaces, hyphens, underscores, dots
	allowedPattern := regexp.MustCompile(`^[\p{L}\p{N}\s\-_.]+$`)

	// Block common injection patterns
	blockedPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(script|javascript|vbscript)`),
		regexp.MustCompile(`(?i)(<|>|&lt;|&gt;)`),
		regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop)`),
		regexp.MustCompile(`(?i)(eval|exec|system)`),
		regexp.MustCompile(`\$\{.*\}`),   // Template injection
		regexp.MustCompile(`\{\{.*\}\}`), // Template injection
	}

	return &QueryValidator{
		maxLength:       MaxQueryLength,
		minLength:       MinQueryLength,
		allowedPattern:  allowedPattern,
		blockedPatterns: blockedPatterns,
	}
}

// ValidateQuery validates a search query for security and format
func (v *QueryValidator) ValidateQuery(query string) error {
	// Check length
	if len(query) > v.maxLength {
		return ErrQueryTooLong
	}
	if len(query) < v.minLength {
		return ErrQueryTooShort
	}

	// Check for control characters
	for _, r := range query {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return ErrInvalidCharacters
		}
	}

	// Check against blocked patterns
	for _, pattern := range v.blockedPatterns {
		if pattern.MatchString(query) {
			return ErrSuspiciousPattern
		}
	}

	// Check against allowed pattern
	if !v.allowedPattern.MatchString(query) {
		return ErrInvalidCharacters
	}

	return nil
}

// SanitizeQuery sanitizes a query string
func (v *QueryValidator) SanitizeQuery(query string) string {
	// Trim whitespace
	query = strings.TrimSpace(query)

	// Remove multiple spaces
	query = regexp.MustCompile(`\s+`).ReplaceAllString(query, " ")

	// Remove dangerous characters
	query = regexp.MustCompile(`[<>&"']`).ReplaceAllString(query, "")

	return query
}

// ValidateUserID validates user ID format
func ValidateUserID(userID string) error {
	if userID == "" {
		return nil // Optional field
	}

	// UUID format or alphanumeric
	uuidPattern := regexp.MustCompile(`^[a-fA-F0-9-]{8,36}$|^[a-zA-Z0-9_-]{3,50}$`)
	if !uuidPattern.MatchString(userID) {
		return errors.New("invalid user ID format")
	}

	return nil
}

// ValidateSessionID validates session ID format
func ValidateSessionID(sessionID string) error {
	if sessionID == "" {
		return nil // Optional field
	}

	// Alphanumeric with limited special chars
	pattern := regexp.MustCompile(`^[a-zA-Z0-9_-]{10,100}$`)
	if !pattern.MatchString(sessionID) {
		return errors.New("invalid session ID format")
	}

	return nil
}

// ValidateTerm validates suggestion terms
func ValidateTerm(term string) error {
	if len(term) == 0 {
		return errors.New("term cannot be empty")
	}

	if len(term) > 200 {
		return errors.New("term too long")
	}

	// Check for basic injection patterns
	dangerous := regexp.MustCompile(`(?i)(script|javascript|<|>|&lt;|&gt;)`)
	if dangerous.MatchString(term) {
		return errors.New("term contains dangerous content")
	}

	return nil
}
