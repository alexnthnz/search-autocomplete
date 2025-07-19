# 🔍 Search Autocomplete System

A high-performance, scalable search autocomplete system built with Go, featuring real-time suggestions, intelligent ranking, caching, and fuzzy matching capabilities.

## 🌟 Features

### Core Functionality
- **Real-time Autocomplete**: Sub-100ms response times for search suggestions
- **Intelligent Ranking**: Multi-factor scoring based on frequency, recency, and relevance
- **Fuzzy Matching**: Handles typos and common misspellings with Levenshtein distance
- **Prefix Matching**: Efficient Trie-based data structure for fast prefix searches
- **Personalization**: User-specific suggestions based on search history and context
- **Input Validation**: XSS/injection protection with comprehensive query sanitization

### Performance & Scalability
- **High Throughput**: Handles millions of queries with token bucket rate limiting (100 req/s)
- **Multi-level Caching**: Redis + in-memory LRU caching for optimal performance
- **Database Persistence**: PostgreSQL integration for data durability and analytics
- **Async Processing**: Non-blocking data pipeline for real-time updates
- **Memory Optimization**: Compressed Trie structure with efficient memory usage
- **Prometheus Metrics**: Comprehensive monitoring and observability

### Data Processing & Analytics
- **Real-time Analytics**: Live tracking of search patterns and trends
- **Batch Processing**: Efficient bulk updates for suggestion data
- **Trending Detection**: Automatic identification of popular search terms
- **Category Classification**: Automatic categorization of search terms
- **Search Logs**: Persistent logging with user session tracking
- **Performance Metrics**: Query latency, cache hit ratios, and error tracking

### Security & Reliability
- **Input Sanitization**: Protection against XSS, injection, and malicious queries
- **Structured Error Handling**: Custom error types with proper HTTP status codes
- **API Key Authentication**: Secure admin endpoints with header-based auth
- **CORS Configuration**: Configurable cross-origin resource sharing
- **Health Monitoring**: Built-in health checks and service status endpoints
- **Graceful Shutdown**: Proper resource cleanup and connection handling

### Developer Experience
- **RESTful API**: Clean, documented endpoints for easy integration
- **Docker Compose**: One-command local development setup with Redis
- **Integration Tests**: Comprehensive test suite covering all endpoints (18 tests)
- **Admin Interface**: Protected endpoints for suggestion management
- **Web UI**: Interactive frontend for testing and demonstration

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend UI   │    │  Load Balancer  │    │   CDN/Edge      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
┌────────────────────────────────┼──────────────────────────────────┐
│                                │                                  │
│  ┌─────────────────┐    ┌──────▼──────┐    ┌─────────────────┐    │
│  │   Rate Limiter  │    │  API Server │    │   Prometheus    │    │
│  │   (100 req/s)   │    │  (Gin + Auth)│    │   Monitoring    │    │
│  └─────────────────┘    └──────┬──────┘    └─────────────────┘    │
│                                │                                  │
│  ┌─────────────────┐    ┌──────▼──────┐    ┌─────────────────┐    │
│  │  Redis Cache    │◄───┤   Service   ├───►│  Data Pipeline  │    │
│  │  (L2 Cache)     │    │   Layer     │    │  (Analytics)    │    │
│  └─────────────────┘    │ (Validation)│    └──────┬──────────┘    │
│                         └──────┬──────┘           │              │
│  ┌─────────────────┐           │                  ▼              │
│  │   Trie Index    │◄──────────┘          ┌─────────────────┐    │
│  │  (In-Memory)    │                      │   PostgreSQL    │    │
│  └─────────────────┘                      │   Database      │    │
│                                           │ (Persistence)   │    │
│                                           └─────────────────┘    │
└───────────────────────────────────────────────────────────────────┘
```

### Components

1. **API Layer**: HTTP endpoints with Gin framework, JWT/API key auth, rate limiting, and CORS
2. **Service Layer**: Core business logic with input validation and structured error handling
3. **Trie Index**: Thread-safe in-memory prefix tree for fast suggestion retrieval
4. **Cache Layer**: Multi-level caching (Redis L2 + in-memory L1) with intelligent invalidation
5. **Data Pipeline**: Asynchronous processing for search logs, analytics, and trending detection
6. **Database Layer**: PostgreSQL for persistent storage of suggestions and search analytics
7. **Monitoring**: Prometheus metrics, health checks, and comprehensive observability
8. **Security**: Input sanitization, XSS protection, and injection attack prevention

## 🚀 Quick Start

### Prerequisites
- Go 1.21+ 
- Docker & Docker Compose (recommended for local development)
- PostgreSQL (optional, for persistent storage)
- Redis (optional, for distributed caching)
- Make (for build automation)
- curl & jq (for API testing)

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/alexnthnz/search-autocomplete.git
cd search-autocomplete
```

### Quick Start with Docker Compose (Recommended)

2. **Start with Docker Compose** (includes Redis)
```bash
make docker-compose-up
```

This will start:
- Autocomplete service on `http://localhost:8080`
- Redis cache on `localhost:6379`
- Web interface at `http://localhost:8080`

### Manual Installation

2. **Install dependencies**
```bash
make deps
```

3. **Build and run**
```bash
make run
```

The service will start on `http://localhost:8080` with a web interface available at the root URL.

### Alternative Docker Methods

```bash
# Build and run single container
make docker-run

# Stop Docker Compose
make docker-compose-down
```

## 🔒 Security Features

### Input Validation & Sanitization
- **Query Validation**: Prevents XSS, injection attacks, and malicious input
- **Length Limits**: Query length capped at 100 characters
- **Character Filtering**: Only allows alphanumeric, spaces, hyphens, underscores, dots
- **Pattern Blocking**: Blocks script tags, SQL injection, and template injection patterns
- **Unicode Normalization**: Proper handling of international characters

### Authentication & Authorization
- **API Key Authentication**: Secure admin endpoints with `X-API-Key` header
- **Rate Limiting**: Token bucket algorithm (100 req/s with 200 burst)
- **CORS Configuration**: Configurable cross-origin resource sharing
- **Input Sanitization**: Automatic cleanup of dangerous characters

### Error Handling
- **Structured Errors**: Custom error types with proper HTTP status codes
- **Security-first**: No sensitive information leaked in error responses
- **Graceful Degradation**: Fallback behavior when services are unavailable

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `API_KEY` | - | API key for admin endpoints (required for production) |
| `ENABLE_CORS` | `true` | Enable CORS headers |
| `CACHE_ENABLED` | `true` | Enable caching layer |
| `CACHE_TTL` | `5m` | Cache time-to-live |
| `REDIS_ENABLED` | `false` | Use Redis for distributed caching |
| `REDIS_HOST` | `localhost` | Redis server host |
| `REDIS_PORT` | `6379` | Redis server port |
| `REDIS_PASSWORD` | - | Redis password (if required) |
| `ENABLE_FUZZY` | `true` | Enable fuzzy matching |
| `FUZZY_THRESHOLD` | `2` | Levenshtein distance threshold |
| `MAX_SUGGESTIONS` | `10` | Maximum suggestions per query |
| `POSTGRES_ENABLED` | `false` | Enable PostgreSQL persistence |
| `POSTGRES_HOST` | `localhost` | PostgreSQL server host |
| `POSTGRES_PORT` | `5432` | PostgreSQL server port |
| `POSTGRES_USER` | `postgres` | PostgreSQL username |
| `POSTGRES_PASSWORD` | - | PostgreSQL password |
| `POSTGRES_DB` | `autocomplete` | PostgreSQL database name |
| `METRICS_ENABLED` | `true` | Enable Prometheus metrics |

### Configuration File

Copy `configs/config.env` to customize settings:

```bash
cp configs/config.env .env
# Edit .env with your configuration
export $(cat .env | grep -v '^#' | xargs)
./bin/autocomplete-server
```

## 📡 API Reference

### Public Endpoints

#### GET /api/v1/autocomplete
Get autocomplete suggestions for a query.

**Parameters:**
- `q` (required): Search query
- `limit` (optional): Maximum number of suggestions (default: 10, max: 50)
- `user_id` (optional): User ID for personalization
- `session_id` (optional): Session ID for personalization

**Example:**
```bash
curl "http://localhost:8080/api/v1/autocomplete?q=app&limit=5"
```

**Response:**
```json
{
  "query": "app",
  "suggestions": [
    {
      "term": "app",
      "frequency": 1200,
      "score": 2400,
      "category": "tech",
      "updated_at": "2024-01-15T10:30:00Z"
    },
    {
      "term": "application",
      "frequency": 800,
      "score": 1600,
      "category": "tech", 
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "latency": "2.5ms",
  "source": "cache"
}
```

#### POST /api/v1/autocomplete
Alternative POST endpoint for complex queries.

**Request Body:**
```json
{
  "query": "search term",
  "limit": 10,
  "user_id": "user123",
  "session_id": "session456"
}
```

#### GET /api/v1/health
Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0"
}
```

#### GET /api/v1/stats
Service statistics and metrics.

**Response:**
```json
{
  "service": {
    "TotalQueries": 1500,
    "CacheHits": 1200,
    "CacheMisses": 300,
    "AvgLatency": 2500000,
    "TrieQueries": 300,
    "FuzzyQueries": 50
  },
  "trie": {
    "suggestions_count": 10000
  },
  "uptime": "2h30m15s"
}
```

#### GET /metrics
Prometheus metrics endpoint (if metrics enabled).

**Response:**
```
# HELP autocomplete_requests_total Total number of autocomplete requests
# TYPE autocomplete_requests_total counter
autocomplete_requests_total{method="GET",endpoint="/api/v1/autocomplete",status="200"} 1250
autocomplete_cache_hits_total{cache_type="redis"} 800
autocomplete_trie_size 10000
...
```

### Admin Endpoints (API Key Required)

Include `X-API-Key` header with your API key.

#### POST /api/v1/admin/suggestions
Add a new suggestion.

**Request Body:**
```json
{
  "term": "machine learning",
  "frequency": 1500,
  "score": 1500,
  "category": "tech"
}
```

#### POST /api/v1/admin/suggestions/batch
Add multiple suggestions at once.

**Request Body:**
```json
[
  {
    "term": "artificial intelligence",
    "frequency": 1200,
    "score": 1200,
    "category": "tech"
  },
  {
    "term": "data science", 
    "frequency": 1000,
    "score": 1000,
    "category": "tech"
  }
]
```

#### PUT /api/v1/admin/suggestions/{term}/frequency
Update the frequency of a suggestion.

**Parameters:**
- `frequency`: New frequency value

#### DELETE /api/v1/admin/suggestions/{term}
Delete a suggestion.

## 🧪 Testing

### Comprehensive Test Suite
The project includes **18 integration tests** covering all endpoints and functionality:

```bash
# Run all tests (unit + integration)
make test

# Run integration tests specifically  
go test ./test/ -v

# Run unit tests for specific components
go test ./internal/trie/ -v
go test ./pkg/utils/ -v

# Run tests with coverage
make test-coverage

# Run benchmarks
make benchmark
```

### Test Coverage
- ✅ **API Endpoints** (18 tests): All GET/POST/PUT/DELETE endpoints
- ✅ **Authentication**: API key validation and unauthorized access
- ✅ **Input Validation**: XSS protection and malicious input handling  
- ✅ **Rate Limiting**: Request throttling and burst handling
- ✅ **Caching**: Cache effectiveness and invalidation
- ✅ **CORS**: Cross-origin request handling
- ✅ **Error Handling**: Proper HTTP status codes and error messages
- ✅ **Fuzzy Search**: Typo tolerance and similarity matching

### API Testing
```bash
# Test all endpoints with sample data
make test-api

# Load sample data for manual testing
make load-sample-data

# Run stress test (100 concurrent requests)
make stress-test
```

### Manual Testing

1. **Start the server**:
```bash
make dev
```

2. **Open the web interface**: Visit `http://localhost:8080`

3. **Try some searches**: Type "app", "coding", "machine" to see suggestions

## 🔥 Performance Optimizations

### 1. Debouncing
The frontend implements 150ms debouncing to reduce API calls during typing.

### 2. Caching Strategy
- **L1 Cache**: In-memory LRU cache for hot queries
- **L2 Cache**: Redis for distributed caching
- **Cache Warming**: Pre-loads popular queries on startup

### 3. Trie Optimizations
- **Path Compression**: Reduces memory usage for sparse branches
- **Concurrent Access**: Read-write mutexes for thread safety
- **Memory Pooling**: Reuses node objects to reduce GC pressure

### 4. Rate Limiting
- **Token Bucket Algorithm**: 100 requests/second with burst of 200
- **Per-IP Limiting**: Prevents abuse from single sources
- **Graceful Degradation**: Returns cached results when under load

### 5. Connection Pooling
- **HTTP Keep-Alive**: Reuses connections for better performance
- **Redis Pooling**: Connection pooling for cache operations

## 📊 Monitoring & Observability

### Prometheus Metrics
Comprehensive metrics collection for production monitoring:

- **Request Metrics**: Total requests, duration histograms, active requests
- **Cache Metrics**: Hit/miss ratios, operation latency by cache type
- **Trie Metrics**: Search operations, insertions, deletions, size tracking
- **Pipeline Metrics**: Processing latency, queue size, throughput
- **Error Metrics**: Error counts by type and component
- **Fuzzy Search**: Usage patterns and match rates

### Available Metrics Endpoints
```bash
# Service statistics (JSON)
curl http://localhost:8080/api/v1/stats

# Prometheus metrics (if enabled)
curl http://localhost:8080/metrics

# Health check
curl http://localhost:8080/api/v1/health
```

### Monitoring Dashboard Setup
```bash
# Example Prometheus configuration
scrape_configs:
  - job_name: 'autocomplete'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### Health Checks
- **API Endpoint Health**: Service availability and response time
- **Cache Connectivity**: Redis/in-memory cache status  
- **Database Status**: PostgreSQL connection health (if enabled)
- **Memory Usage**: Trie size and memory consumption
- **Rate Limiting**: Current request rates and burst capacity

### Structured Logging
JSON-formatted logs with configurable levels and structured fields:
```bash
# Development (verbose logging)
LOG_LEVEL=debug make dev

# Production (error/warn only)
LOG_LEVEL=warn make run-prod

# Log format example
{"level":"info","msg":"HTTP Request","method":"GET","path":"/api/v1/autocomplete","ip":"127.0.0.1","latency":"2.5ms","status":200,"time":"2024-01-15T10:30:00Z"}
```

## 🚀 Deployment

### Local Development
```bash
make dev    # Development mode with hot reload
```

### Production
```bash
make build-prod    # Optimized build
make run-prod      # Production configuration
```

### Docker Deployment
```bash
# Single container
docker build -t search-autocomplete .
docker run -p 8080:8080 search-autocomplete

# With Redis
docker-compose up --build
```

### Environment-Specific Configs

**Development:**
- Debug logging enabled
- CORS enabled for all origins
- In-memory cache fallback
- Sample data auto-loaded
- Integration tests enabled

**Production:**
- Warn-level logging only
- Redis distributed caching
- PostgreSQL persistence
- Prometheus metrics enabled
- API key authentication required
- Rate limiting enforced (100 req/s)
- Input validation and sanitization

### Production Deployment Checklist
- [ ] Set `API_KEY` environment variable
- [ ] Configure PostgreSQL connection
- [ ] Enable Redis for distributed caching
- [ ] Set `LOG_LEVEL=warn` or `LOG_LEVEL=error`
- [ ] Configure Prometheus metrics collection
- [ ] Set up health check monitoring
- [ ] Configure CORS for specific origins
- [ ] Enable HTTPS/TLS termination
- [ ] Set resource limits (CPU/Memory)
- [ ] Configure log aggregation (ELK/Fluentd)

## 🤝 Contributing

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Make your changes**: Follow the existing code style
4. **Add tests**: Ensure good test coverage
5. **Commit your changes**: `git commit -m 'Add amazing feature'`
6. **Push to branch**: `git push origin feature/amazing-feature`
7. **Open a Pull Request**

### Code Style
- Run `make fmt` to format code
- Run `make lint` for linting
- Follow Go best practices
- Add documentation for public APIs

## 📈 Roadmap

### Recently Implemented ✅
- ✅ **Security & Validation**: XSS protection, input sanitization, structured errors
- ✅ **Database Persistence**: PostgreSQL integration with analytics tables
- ✅ **Prometheus Metrics**: Comprehensive monitoring and observability
- ✅ **Docker Compose**: One-command development setup with Redis
- ✅ **Integration Tests**: 18 comprehensive tests covering all endpoints
- ✅ **Rate Limiting**: Token bucket algorithm (100 req/s with burst)
- ✅ **Structured Logging**: JSON-formatted logs with configurable levels

### Planned Features
- [ ] **Machine Learning Integration**: ML-based ranking models
- [ ] **Multi-language Support**: Unicode normalization and international queries
- [ ] **Analytics Dashboard**: Web-based monitoring interface with charts
- [ ] **A/B Testing Framework**: Experiment with different ranking algorithms
- [ ] **Geolocation Awareness**: Location-based suggestions
- [ ] **Advanced Personalization**: User behavior modeling with ML
- [ ] **Voice Search Support**: Audio input processing
- [ ] **JWT Authentication**: Replace API key auth with JWT tokens
- [ ] **GraphQL API**: Alternative to REST endpoints

### Performance Improvements  
- [ ] **Distributed Trie**: Sharded across multiple nodes
- [ ] **GPU Acceleration**: CUDA-based similarity matching
- [ ] **Edge Computing**: Deploy to CDN edge locations
- [ ] **Streaming Updates**: Real-time suggestion updates via WebSocket
- [ ] **Circuit Breaker**: Fault tolerance for external dependencies

## 🐛 Troubleshooting

### Common Issues

**1. Port already in use**
```bash
# Find process using port 8080
lsof -i :8080
# Kill the process
kill -9 <PID>
```

**2. Redis connection failed**
```bash
# Check if Redis is running
redis-cli ping
# Start Redis
redis-server
# Or disable Redis in config
REDIS_ENABLED=false make run
```

**3. High memory usage**
```bash
# Check memory stats
curl http://localhost:8080/api/v1/stats
# Reduce cache TTL
CACHE_TTL=1m make run
```

**4. Slow response times**
```bash
# Enable performance logging
LOG_LEVEL=debug make run
# Check cache hit ratio
make stats
```

### Performance Tuning

**For High Traffic:**
- Enable Redis caching
- Increase rate limiting thresholds
- Use load balancer with multiple instances
- Monitor cache hit ratios

**For Memory Optimization:**
- Reduce cache TTL
- Limit maximum suggestions
- Enable trie compression
- Monitor GC metrics

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **Gin Web Framework**: HTTP server and middleware
- **Redis**: High-performance caching
- **Logrus**: Structured logging
- **Testify**: Testing framework

## 📞 Support

- **Documentation**: This README and inline code comments
- **Issues**: [GitHub Issues](https://github.com/alexnthnz/search-autocomplete/issues)
- **Discussions**: [GitHub Discussions](https://github.com/alexnthnz/search-autocomplete/discussions)

---

Built with ❤️ by [Alex Nguyen](https://github.com/alexnthnz)