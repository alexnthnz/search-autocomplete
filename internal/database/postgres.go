package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"github.com/alexnthnz/search-autocomplete/pkg/models"
)

// Config holds database configuration
type Config struct {
	Host         string
	Port         int
	User         string
	Password     string
	DatabaseName string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
}

// PostgresDB handles PostgreSQL operations
type PostgresDB struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(config Config, logger *logrus.Logger) (*PostgresDB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DatabaseName, config.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	postgres := &PostgresDB{
		db:     db,
		logger: logger,
	}

	// Initialize schema
	if err := postgres.initSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	logger.Info("Successfully connected to PostgreSQL database")
	return postgres, nil
}

// initSchema creates necessary tables
func (p *PostgresDB) initSchema(ctx context.Context) error {
	schema := `
	-- Suggestions table
	CREATE TABLE IF NOT EXISTS suggestions (
		id SERIAL PRIMARY KEY,
		term VARCHAR(200) NOT NULL UNIQUE,
		frequency BIGINT NOT NULL DEFAULT 0,
		score DOUBLE PRECISION NOT NULL DEFAULT 0,
		category VARCHAR(50),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	-- Search logs table
	CREATE TABLE IF NOT EXISTS search_logs (
		id SERIAL PRIMARY KEY,
		query VARCHAR(200) NOT NULL,
		user_id VARCHAR(100),
		session_id VARCHAR(100),
		ip_address INET,
		user_agent TEXT,
		timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		response_time_ms INTEGER,
		result_count INTEGER,
		source VARCHAR(20) -- 'cache', 'trie', 'fuzzy'
	);

	-- Analytics table for aggregated data
	CREATE TABLE IF NOT EXISTS query_analytics (
		id SERIAL PRIMARY KEY,
		query VARCHAR(200) NOT NULL,
		date DATE NOT NULL,
		hour INTEGER CHECK (hour >= 0 AND hour <= 23),
		search_count INTEGER NOT NULL DEFAULT 0,
		unique_users INTEGER NOT NULL DEFAULT 0,
		avg_response_time_ms DOUBLE PRECISION,
		PRIMARY KEY (query, date, hour)
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_suggestions_term ON suggestions(term);
	CREATE INDEX IF NOT EXISTS idx_suggestions_frequency ON suggestions(frequency DESC);
	CREATE INDEX IF NOT EXISTS idx_suggestions_category ON suggestions(category);
	CREATE INDEX IF NOT EXISTS idx_suggestions_updated_at ON suggestions(updated_at DESC);

	CREATE INDEX IF NOT EXISTS idx_search_logs_query ON search_logs(query);
	CREATE INDEX IF NOT EXISTS idx_search_logs_timestamp ON search_logs(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_search_logs_user_id ON search_logs(user_id);
	CREATE INDEX IF NOT EXISTS idx_search_logs_session_id ON search_logs(session_id);

	CREATE INDEX IF NOT EXISTS idx_query_analytics_query ON query_analytics(query);
	CREATE INDEX IF NOT EXISTS idx_query_analytics_date ON query_analytics(date DESC);

	-- Function to update updated_at automatically
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = CURRENT_TIMESTAMP;
		RETURN NEW;
	END;
	$$ language 'plpgsql';

	-- Trigger for suggestions table
	DROP TRIGGER IF EXISTS update_suggestions_updated_at ON suggestions;
	CREATE TRIGGER update_suggestions_updated_at
		BEFORE UPDATE ON suggestions
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();
	`

	_, err := p.db.ExecContext(ctx, schema)
	return err
}

// StoreSuggestion stores a suggestion in the database
func (p *PostgresDB) StoreSuggestion(ctx context.Context, suggestion models.Suggestion) error {
	query := `
		INSERT INTO suggestions (term, frequency, score, category) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (term) 
		DO UPDATE SET 
			frequency = $2,
			score = $3,
			category = $4,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := p.db.ExecContext(ctx, query, suggestion.Term, suggestion.Frequency, suggestion.Score, suggestion.Category)
	if err != nil {
		p.logger.WithError(err).WithField("term", suggestion.Term).Error("Failed to store suggestion")
		return fmt.Errorf("failed to store suggestion: %w", err)
	}

	return nil
}

// GetSuggestions retrieves suggestions by prefix
func (p *PostgresDB) GetSuggestions(ctx context.Context, prefix string, limit int) ([]models.Suggestion, error) {
	query := `
		SELECT term, frequency, score, category, updated_at
		FROM suggestions 
		WHERE term ILIKE $1 || '%'
		ORDER BY score DESC, frequency DESC
		LIMIT $2
	`

	rows, err := p.db.QueryContext(ctx, query, prefix, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query suggestions: %w", err)
	}
	defer rows.Close()

	var suggestions []models.Suggestion
	for rows.Next() {
		var s models.Suggestion
		err := rows.Scan(&s.Term, &s.Frequency, &s.Score, &s.Category, &s.UpdatedAt)
		if err != nil {
			p.logger.WithError(err).Error("Failed to scan suggestion")
			continue
		}
		suggestions = append(suggestions, s)
	}

	return suggestions, rows.Err()
}

// GetAllSuggestions retrieves all suggestions for loading into memory
func (p *PostgresDB) GetAllSuggestions(ctx context.Context) ([]models.Suggestion, error) {
	query := `
		SELECT term, frequency, score, category, updated_at
		FROM suggestions 
		ORDER BY frequency DESC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all suggestions: %w", err)
	}
	defer rows.Close()

	var suggestions []models.Suggestion
	for rows.Next() {
		var s models.Suggestion
		err := rows.Scan(&s.Term, &s.Frequency, &s.Score, &s.Category, &s.UpdatedAt)
		if err != nil {
			p.logger.WithError(err).Error("Failed to scan suggestion")
			continue
		}
		suggestions = append(suggestions, s)
	}

	return suggestions, rows.Err()
}

// UpdateSuggestionFrequency updates the frequency of a suggestion
func (p *PostgresDB) UpdateSuggestionFrequency(ctx context.Context, term string, frequency int64) error {
	query := `
		UPDATE suggestions 
		SET frequency = $2, score = $2, updated_at = CURRENT_TIMESTAMP
		WHERE term = $1
	`

	result, err := p.db.ExecContext(ctx, query, term, frequency)
	if err != nil {
		return fmt.Errorf("failed to update suggestion frequency: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("suggestion not found: %s", term)
	}

	return nil
}

// DeleteSuggestion removes a suggestion
func (p *PostgresDB) DeleteSuggestion(ctx context.Context, term string) error {
	query := `DELETE FROM suggestions WHERE term = $1`

	result, err := p.db.ExecContext(ctx, query, term)
	if err != nil {
		return fmt.Errorf("failed to delete suggestion: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("suggestion not found: %s", term)
	}

	return nil
}

// LogSearch stores a search log entry
func (p *PostgresDB) LogSearch(ctx context.Context, log models.SearchLog, responseTimeMs int, resultCount int, source string) error {
	query := `
		INSERT INTO search_logs (query, user_id, session_id, ip_address, timestamp, response_time_ms, result_count, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := p.db.ExecContext(ctx, query,
		log.Query, log.UserID, log.SessionID, log.IPAddress,
		log.Timestamp, responseTimeMs, resultCount, source)

	if err != nil {
		p.logger.WithError(err).Error("Failed to log search")
		return fmt.Errorf("failed to log search: %w", err)
	}

	return nil
}

// GetSearchAnalytics retrieves search analytics
func (p *PostgresDB) GetSearchAnalytics(ctx context.Context, query string, days int) ([]SearchAnalytic, error) {
	sqlQuery := `
		SELECT query, date, hour, search_count, unique_users, avg_response_time_ms
		FROM query_analytics 
		WHERE query = $1 AND date >= CURRENT_DATE - INTERVAL '%d days'
		ORDER BY date DESC, hour DESC
	`

	rows, err := p.db.QueryContext(ctx, fmt.Sprintf(sqlQuery, days), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query analytics: %w", err)
	}
	defer rows.Close()

	var analytics []SearchAnalytic
	for rows.Next() {
		var a SearchAnalytic
		err := rows.Scan(&a.Query, &a.Date, &a.Hour, &a.SearchCount, &a.UniqueUsers, &a.AvgResponseTimeMs)
		if err != nil {
			p.logger.WithError(err).Error("Failed to scan analytics")
			continue
		}
		analytics = append(analytics, a)
	}

	return analytics, rows.Err()
}

// GetTopQueries retrieves most popular queries
func (p *PostgresDB) GetTopQueries(ctx context.Context, limit int, days int) ([]QueryStats, error) {
	query := `
		SELECT query, COUNT(*) as search_count, COUNT(DISTINCT user_id) as unique_users
		FROM search_logs 
		WHERE timestamp >= CURRENT_DATE - INTERVAL '%d days'
		GROUP BY query
		ORDER BY search_count DESC
		LIMIT $1
	`

	rows, err := p.db.QueryContext(ctx, fmt.Sprintf(query, days), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top queries: %w", err)
	}
	defer rows.Close()

	var stats []QueryStats
	for rows.Next() {
		var s QueryStats
		err := rows.Scan(&s.Query, &s.SearchCount, &s.UniqueUsers)
		if err != nil {
			p.logger.WithError(err).Error("Failed to scan query stats")
			continue
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// SearchAnalytic represents search analytics data
type SearchAnalytic struct {
	Query             string    `json:"query"`
	Date              time.Time `json:"date"`
	Hour              int       `json:"hour"`
	SearchCount       int       `json:"search_count"`
	UniqueUsers       int       `json:"unique_users"`
	AvgResponseTimeMs float64   `json:"avg_response_time_ms"`
}

// QueryStats represents query statistics
type QueryStats struct {
	Query       string `json:"query"`
	SearchCount int    `json:"search_count"`
	UniqueUsers int    `json:"unique_users"`
}
