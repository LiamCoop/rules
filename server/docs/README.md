# Multi-Tenant Rules Engine

A high-performance, multi-tenant rules engine built with Go and PostgreSQL, supporting dynamic schemas and CEL (Common Expression Language) rule evaluation.

## Performance Benchmarks

⚡ **Load Tested & Production Ready**
- **500+ requests/second** sustained throughput
- **0% error rate** under stress testing (250 concurrent users)
- **P50 latency**: 197ms
- **P95 latency**: 326ms
- **Multi-tenant**: Supports 10+ concurrent tenants with isolated data

*Validated with k6 load testing against Railway deployment (see `loadtests/` for test suite)*

## Features

- 🏢 **Multi-tenant architecture** with complete data isolation
- 📊 **Dynamic schema definition** per tenant
- 🔧 **CEL rule engine** for flexible business logic
- ⚡ **Real-time rule evaluation** with <200ms P50 latency
- 🔄 **Zero-downtime schema updates** using versioning
- 🐳 **Containerized deployment** with Docker
- 📈 **Production-grade load testing** with k6

## Quick Start

### Prerequisites

- Go 1.24+
- PostgreSQL 16+
- Docker (optional, for deployment)

### Local Development

```bash
# Install dependencies
go mod download

# Set up database
export DATABASE_URL="postgresql://user:pass@localhost/rules?sslmode=disable"

# Run migrations
psql $DATABASE_URL -f migrations/000001_initial_schema.up.sql

# Start server
go run ./cmd/server/main.go
```

Server will be available at `http://localhost:8080`

### Docker Deployment

```bash
# Build image
docker build -t rules-engine .

# Run container
docker run -p 8080:8080 \
  -e DATABASE_URL="postgresql://..." \
  rules-engine
```

## API Documentation

Full API documentation available at:
- **Swagger UI**: `http://localhost:8080/swagger/`
- **API Guide**: [API.md](./API.md)

### Quick Example

```bash
# Create a tenant
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name":"acme-corp"}'

# Create schema
curl -X POST http://localhost:8080/api/v1/tenants/{tenantId}/schema \
  -H "Content-Type: application/json" \
  -d '{
    "definition": {
      "facts": {
        "age": "int",
        "country": "string",
        "is_premium": "bool"
      }
    }
  }'

# Create rule
curl -X POST http://localhost:8080/api/v1/tenants/{tenantId}/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Adult Check",
    "expression": "facts.age >= 18",
    "description": "User must be 18 or older"
  }'

# Evaluate rules
curl -X POST http://localhost:8080/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "{tenantId}",
    "facts": {
      "age": 25,
      "country": "US",
      "is_premium": true
    }
  }'
```

## Load Testing

Comprehensive load testing suite included using k6:

```bash
cd loadtests

# Seed test data (one-time)
k6 run seed.js

# Run baseline load test (9.5 minutes, 100 VUs)
k6 run load-test.js

# Run stress test (5 minutes, 250 VUs peak)
k6 run stress-test.js
```

See [loadtests/README.md](./loadtests/README.md) for detailed testing documentation.

## Architecture

### Current Stack

- **Language**: Go 1.24 with Chi router
- **Database**: PostgreSQL 16 with JSONB support
- **Rule Engine**: Google CEL for sandboxed expression evaluation
- **Deployment**: Railway (Docker containers)

### Database Schema

- **tenants**: Multi-tenant isolation (UUID primary keys)
- **schemas**: Dynamic schema definitions per tenant (versioned, JSONB)
- **rules**: CEL expressions with active/inactive state
- **derived_fields**: Computed fields with dependency tracking
- **schema_changelog**: Audit trail for schema changes

### Key Design Decisions

1. **PostgreSQL over NoSQL**: ACID compliance, strong schema validation, efficient indexing
2. **UUID primary keys**: Better for distributed systems, no auto-increment collisions
3. **JSONB for schemas**: Flexible schema definitions with query performance
4. **Versioned schemas**: Zero-downtime updates via version tracking
5. **CEL for rules**: Sandboxed, type-safe, Google-maintained

---

## Scaling Architecture

### Current Performance Baseline

Through k6 load testing against Railway deployment:
- **Throughput**: 500+ RPS sustained (250 concurrent users)
- **Error Rate**: 0.00% under stress
- **Bottleneck Identified**: PostgreSQL at 6.5 vCPU vs Go service at 2.8 vCPU

**Key Insight**: The Go service has significant headroom (2.3x less CPU usage than PostgreSQL), making the database the primary scaling constraint.

### Horizontal Scaling Strategies

#### 1. Database Sharding by Tenant (Primary Strategy)

**Approach**: Partition tenants across multiple PostgreSQL instances based on tenant ID hash.

**Benefits**:
- **Linear scaling**: 3 database instances = ~1,500 RPS (3x current capacity)
- Natural isolation boundary (each tenant already isolated)
- No cross-shard queries needed for evaluation
- Predictable performance per shard

**Implementation**:
```go
// Shard routing logic
func getShardForTenant(tenantID string) *sql.DB {
    shardIndex := hash(tenantID) % numShards
    return dbShards[shardIndex]
}
```

**Trade-offs**:
- Requires tenant-to-shard mapping
- Cross-tenant analytics becomes complex
- Schema migrations across shards

**Estimated Capacity**: 1,500+ RPS with 3 shards

---

#### 2. Read Replicas for Hot Tenants

**Approach**: Identify high-traffic tenants (80/20 rule) and route their read queries to dedicated replicas.

**Benefits**:
- Offload read traffic from primary database
- Simple to implement with PostgreSQL streaming replication
- No application logic changes for writes

**Implementation**:
- Monitor tenant request rates
- Create read replicas for top 20% of tenants
- Route `GET /rules` and evaluation reads to replicas
- Keep writes on primary

**Trade-offs**:
- Replication lag (eventual consistency)
- More infrastructure to manage
- Limited to read-heavy workloads

**Estimated Capacity**: 2x throughput for read-heavy tenants (~1,000 RPS)

---

#### 3. Redis Caching Layer (Quick Win)

**Approach**: Cache compiled CEL rules in Redis to reduce database load.

**Benefits**:
- **40-60% reduction in DB load** (rule compilation is CPU-intensive)
- Sub-millisecond cache hits
- Easy to implement with go-redis

**Implementation**:
```go
// Cache compiled rules
cacheKey := fmt.Sprintf("rule:%s:compiled", ruleID)
redis.Set(ctx, cacheKey, compiledRule, 5*time.Minute)

// Evaluation flow
if cached := redis.Get(ctx, cacheKey); cached != nil {
    return evaluateWithCachedRule(cached, facts)
}
// Fall back to DB + compile
```

**Cache Invalidation**:
- TTL-based: 5-10 minute expiry
- Event-based: Invalidate on rule update/delete

**Trade-offs**:
- Additional infrastructure (Redis instance)
- Cache invalidation complexity
- Memory costs for hot rules

**Estimated Capacity**: 800-1,000 RPS (60% DB load reduction on current hardware)

---

#### 4. Horizontal Scaling of Go Service

**Approach**: Deploy 3-4 Go instances behind a load balancer.

**Benefits**:
- Utilizes existing CPU headroom (currently at 2.8 vCPU)
- Fault tolerance (instance failures don't bring down service)
- Simple to implement on Railway

**Implementation**:
- Railway automatic scaling or manual scaling
- Load balancer distributes traffic
- Stateless service design (no sticky sessions needed)

**Trade-offs**:
- Doesn't solve database bottleneck
- Increased infrastructure costs
- Requires load balancer setup

**Estimated Capacity**: 2,000+ RPS (4 instances × 500 RPS each)

**Best Combined With**: Database sharding or caching

---

#### 5. Connection Pool Tuning (Low-Hanging Fruit)

**Approach**: Optimize PostgreSQL connection pool settings for high concurrency.

**Current Settings** (likely defaults):
```go
db.SetMaxOpenConns(25)  // Default
db.SetMaxIdleConns(25)
db.SetConnMaxLifetime(5 * time.Minute)
```

**Optimized Settings**:
```go
db.SetMaxOpenConns(100)  // Increase for higher concurrency
db.SetMaxIdleConns(50)   // Keep connections warm
db.SetConnMaxLifetime(30 * time.Minute)
db.SetConnMaxIdleTime(10 * time.Minute)
```

**PostgreSQL Side**:
```sql
-- Increase max connections
ALTER SYSTEM SET max_connections = 200;

-- Tune work_mem for rule compilation queries
ALTER SYSTEM SET work_mem = '32MB';
```

**Estimated Improvement**: 10-20% throughput increase (550-600 RPS)

---

### Recommended Scaling Path

**Phase 1: Quick Wins (1-2 weeks)**
1. ✅ Connection pool tuning (10-20% gain)
2. ✅ Redis caching for compiled rules (40-60% DB load reduction)
3. ✅ Horizontal scaling of Go service (2-3 instances)

**Expected Capacity**: 800-1,000 RPS

---

**Phase 2: Architectural (1-2 months)**
1. ✅ Database sharding by tenant ID hash (3 shards initially)
2. ✅ Read replicas for high-traffic tenants
3. ✅ Load balancer with health checks

**Expected Capacity**: 1,500-2,000 RPS

---

**Phase 3: Advanced (3-6 months)**
1. ✅ Multi-region deployment for global latency
2. ✅ Event-driven architecture for async rule processing
3. ✅ ML-based tenant routing to optimal shards
4. ✅ GraphQL API for complex cross-tenant analytics

**Expected Capacity**: 5,000+ RPS

---

### Cost vs Performance Trade-offs

| Strategy | Complexity | Cost Impact | RPS Gain | Time to Implement |
|----------|-----------|-------------|----------|-------------------|
| Connection Pool Tuning | Low | None | +10-20% | 1 day |
| Redis Caching | Medium | +$15/mo | +40-60% | 1 week |
| Horizontal Scaling (Go) | Low | +$20/mo/instance | Linear | 2 days |
| Database Sharding | High | +$50/mo/shard | Linear (3x) | 4-6 weeks |
| Read Replicas | Medium | +$50/mo/replica | +2x reads | 1-2 weeks |

---

### Monitoring & Observability

**Key Metrics to Track**:
- Request rate per tenant (identify hot tenants)
- PostgreSQL CPU usage (scale trigger)
- Cache hit rate (Redis effectiveness)
- P95/P99 latency (user experience)
- Error rate by tenant (isolation validation)

**Recommended Tools**:
- Railway metrics dashboard (current)
- Prometheus + Grafana (advanced)
- DataDog or New Relic (production)

---

## Testing

### Unit Tests

```bash
# Run all unit tests
make test

# With race detection
make test-race

# With coverage
make test-coverage
```

### Integration Tests

```bash
# Requires Docker for testcontainers
make test-integration

# All tests
make test-all
```

### Load Tests

See [loadtests/README.md](./loadtests/README.md) for comprehensive load testing guide.

## Project Structure

```
.
├── cmd/
│   └── server/          # Main application entry point
│       ├── main.go      # Server setup and routing
│       └── models.go    # Request/response models
├── rules/               # CEL rule engine
│   ├── engine.go        # Rule compilation and evaluation
│   ├── store.go         # Database operations
│   └── types.go         # Type definitions
├── multitenantengine/   # Multi-tenant management
│   ├── manager.go       # Tenant engine manager
│   └── validation.go    # Schema validation
├── migrations/          # Database migrations
│   └── 000001_initial_schema.up.sql
├── loadtests/           # k6 load testing suite
│   ├── seed.js          # Database seeding
│   ├── load-test.js     # Baseline load test
│   └── stress-test.js   # Stress/peak load test
├── Dockerfile           # Container image definition
├── Makefile            # Build and test automation
└── README.md           # This file
```

## Contributing

This is a personal project for learning and portfolio purposes. If you'd like to use it or suggest improvements, feel free to open an issue!

## License

MIT License - feel free to use this code for learning or as a starting point for your own projects.

---

## Additional Resources

- [API Documentation](./API.md) - Complete API reference
- [Load Testing Guide](./loadtests/README.md) - k6 testing instructions
- [Performance Achievements](./brag.md) - Benchmarks and resume bullet points
- [Migration Guide](./MIGRATIONS.md) - Database migration documentation

---

**Built with ❤️ using Go, PostgreSQL, and k6**
