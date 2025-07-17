# 🔍 Search Autocomplete System

A high-performance, scalable search autocomplete system built with Go, featuring real-time suggestions, intelligent ranking, caching, and fuzzy matching capabilities.

## 🌟 Features

### Core Functionality
- **Real-time Autocomplete**: Sub-100ms response times for search suggestions
- **Intelligent Ranking**: Multi-factor scoring based on frequency, recency, and relevance
- **Fuzzy Matching**: Handles typos and common misspellings
- **Prefix Matching**: Efficient Trie-based data structure for fast prefix searches
- **Personalization**: User-specific suggestions based on search history (optional)

### Performance & Scalability
- **High Throughput**: Handles millions of queries with rate limiting
- **Multi-level Caching**: Redis + in-memory caching for optimal performance
- **Horizontal Scaling**: Shardable architecture for large datasets
- **Async Processing**: Non-blocking data pipeline for real-time updates
- **Memory Optimization**: Compressed Trie structure with efficient memory usage

### Data Processing
- **Real-time Analytics**: Live tracking of search patterns and trends
- **Batch Processing**: Efficient bulk updates for suggestion data
- **Trending Detection**: Automatic identification of popular search terms
- **Category Classification**: Automatic categorization of search terms
- **Frequency Updates**: Dynamic scoring based on search volume

### API & Integration
- **RESTful API**: Clean, documented endpoints for easy integration
- **Rate Limiting**: Configurable request throttling
- **CORS Support**: Cross-origin resource sharing for web applications
- **Health Monitoring**: Built-in health checks and metrics
- **Admin Interface**: Protected endpoints for suggestion management

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
│  │   Rate Limiter  │    │  API Server │    │   Monitoring    │    │
│  └─────────────────┘    └──────┬──────┘    └─────────────────┘    │
│                                │                                  │
│  ┌─────────────────┐    ┌──────▼──────┐    ┌─────────────────┐    │
│  │  Redis Cache    │◄───┤   Service   ├───►│  Data Pipeline  │    │
│  └─────────────────┘    │   Layer     │    └─────────────────┘    │
│                         └──────┬──────┘                           │
│  ┌─────────────────┐           │                                  │
│  │   Trie Index    │◄──────────┘                                  │
│  └─────────────────┘                                              │
└───────────────────────────────────────────────────────────────────┘
```

### Components

1. **API Layer**: HTTP endpoints with middleware for authentication, rate limiting, and CORS
2. **Service Layer**: Core business logic for autocomplete functionality
3. **Trie Index**: In-memory prefix tree for fast suggestion retrieval
4. **Cache Layer**: Redis + in-memory caching for frequently accessed data
5. **Data Pipeline**: Asynchronous processing for search logs and trending analysis
6. **Monitoring**: Health checks, metrics collection, and performance tracking

## 🚀 Quick Start

### Prerequisites
- Go 1.21+ 
- Redis (optional, uses in-memory cache by default)
- Make (for build automation)

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/alexnthnz/search-autocomplete.git
cd search-autocomplete
```

2. **Install dependencies**
```bash
make deps
```

3. **Build and run**
```bash
make run
```

The service will start on `http://localhost:8080` with a web interface available at the root URL.

### Using Docker

```bash
# Build and run with Docker
make docker-run

# Or use docker-compose (includes Redis)
make docker-compose-up
```

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `API_KEY` | - | API key for admin endpoints |
| `ENABLE_CORS` | `true` | Enable CORS headers |
| `CACHE_ENABLED` | `true` | Enable caching layer |
| `CACHE_TTL` | `5m` | Cache time-to-live |
| `REDIS_ENABLED` | `false` | Use Redis for caching |
| `REDIS_HOST` | `localhost` | Redis server host |
| `REDIS_PORT` | `6379` | Redis server port |
| `ENABLE_FUZZY` | `true` | Enable fuzzy matching |
| `FUZZY_THRESHOLD` | `2` | Levenshtein distance threshold |
| `MAX_SUGGESTIONS` | `10` | Maximum suggestions per query |

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

### Running Tests
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run benchmarks
make benchmark
```

### API Testing
```bash
# Test all endpoints
make test-api

# Load sample data
make load-sample-data

# Run stress test
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

### Metrics Collected
- Query latency (p50, p95, p99)
- Cache hit ratio
- Request rate and error rate
- Memory usage and GC statistics
- Trending queries and patterns

### Health Checks
- API endpoint health
- Cache connectivity
- Memory usage thresholds
- Response time monitoring

### Logging
Structured JSON logging with configurable levels:
```bash
# Development
LOG_LEVEL=debug make dev

# Production  
LOG_LEVEL=warn make run-prod
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
- CORS enabled
- In-memory cache
- Sample data loaded

**Production:**
- Warn-level logging
- Redis caching
- Rate limiting enforced
- Monitoring enabled

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

### Planned Features
- [ ] **Machine Learning Integration**: ML-based ranking models
- [ ] **Multi-language Support**: Unicode and international queries
- [ ] **Analytics Dashboard**: Web-based monitoring interface
- [ ] **A/B Testing Framework**: Experiment with different ranking algorithms
- [ ] **Geolocation Awareness**: Location-based suggestions
- [ ] **Advanced Personalization**: User behavior modeling
- [ ] **Voice Search Support**: Audio input processing
- [ ] **Federated Search**: Integration with external data sources

### Performance Improvements
- [ ] **Distributed Trie**: Sharded across multiple nodes
- [ ] **GPU Acceleration**: CUDA-based similarity matching
- [ ] **Edge Computing**: Deploy to CDN edge locations
- [ ] **Streaming Updates**: Real-time suggestion updates via WebSocket

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