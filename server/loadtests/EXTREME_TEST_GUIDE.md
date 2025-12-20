# Extreme Load Testing Guide

## Overview

After fixing the database connection pool configuration, we've updated the extreme stress test to find the **real** breaking point of the system.

## What Changed

### Before (Connection Pool Issue)
```go
// Default Go behavior - NOT configured
db.SetMaxIdleConns(2)  // Only 2 idle connections!
```

**Result**: Failed at 2.5k RPS with high RTT despite low CPU

### After (Fixed)
```go
db.SetMaxOpenConns(100)
db.SetMaxIdleConns(50)
db.SetConnMaxLifetime(30 * time.Minute)
db.SetConnMaxIdleTime(10 * time.Minute)
```

**Result**: 6,898 RPS sustained at 3k VUs with only 0.29% error rate

## New Test Profile

### extreme-stress-test.js

**Target**: 15,000 concurrent VUs
**Duration**: ~11 minutes
**Expected Peak RPS**: 15,000+ RPS

**Load Profile**:
```
0s ────→ 1,000 VUs (warm-up)
1m ────→ 3,000 VUs (known good baseline - 6.9k RPS)
2m ────→ 5,000 VUs (first stress level)
3m ────→ 7,500 VUs (medium stress)
4m ────→ 10,000 VUs (serious load)
5m ────→ 12,500 VUs (finding the edge)
6m ────→ 15,000 VUs (MAXIMUM LOAD)
8m ────→ 15,000 VUs (hold at peak)
10m ───→ 0 VUs (cool down)
```

## Running the Test

```bash
cd loadtests

# Make sure you have test data seeded
k6 run seed.js

# Run the extreme test
k6 run extreme-stress-test.js
```

## What to Monitor

### Railway Dashboard (CRITICAL)
Keep the Railway metrics dashboard open during the test:

1. **Application Instance**:
   - CPU usage (watch for 100% utilization)
   - Memory consumption (watch for OOM)
   - Network I/O egress (watch for plateaus)

2. **Database Instance**:
   - CPU usage (likely to max out first)
   - Memory usage
   - Active connections (max = 100 from our config)

### k6 Output
Watch for these key indicators:

1. **Error Rate (`http_req_failed`)**:
   - Stays <1% = Healthy
   - 1-5% = Starting to struggle
   - >10% = Hit a limit

2. **P95 Latency (`http_req_duration{p95}`)**:
   - <500ms = Excellent
   - 500-1000ms = Good under load
   - >2000ms = Degrading

3. **Request Rate (`http_reqs`)**:
   - Should scale linearly with VUs
   - Plateaus = Hit a bottleneck

## Expected Bottlenecks (Ranked)

Based on previous test with low CPU utilization, here's what will likely break first:

### 1. Database Connection Pool (Most Likely)
**Symptom**: Error rate spikes, "too many connections" errors
**When**: Around 10k VUs (100 max connections / avg 10 VUs per connection)
**Fix**: Increase `MaxOpenConns` or add read replicas

### 2. Database CPU (Likely)
**Symptom**: P95 latency increases, but no errors
**When**: When DB CPU hits 100%
**Fix**: Vertical scaling or horizontal with read replicas

### 3. Network Bandwidth (Possible)
**Symptom**: Receiving time increases, throughput plateaus
**When**: >200 Mbps sustained
**Fix**: Horizontal scaling (multiple instances)

### 4. Application CPU (Unlikely)
**Symptom**: App CPU hits 100%
**When**: Very high RPS (>15k)
**Fix**: Horizontal scaling of Go service

## Interpreting Results

### Scenario 1: Handles 15k VUs Successfully
```
✅ Error rate < 5%
✅ P95 latency < 1000ms
✅ All Railway metrics healthy
```

**Conclusion**: Your system is beast mode. Consider pushing to 20k VUs.

### Scenario 2: Breaks at 10k VUs
```
⚠️  Error rate spikes to >10%
⚠️  DB CPU at 100%
```

**Conclusion**: Database is the bottleneck. Next steps:
- Add read replicas
- Implement Redis caching for rules
- Consider database sharding for multi-tenant isolation

### Scenario 3: Breaks at 7.5k VUs
```
⚠️  "Too many connections" errors
⚠️  Connection pool exhaustion
```

**Conclusion**: Hit the 100 connection limit. Options:
- Increase `MaxOpenConns` to 200-300
- Tune PostgreSQL `max_connections`
- Add connection pooling proxy (PgBouncer)

### Scenario 4: Network Bottleneck
```
⚠️  Receiving time >200ms
⚠️  Network I/O plateaus
⚠️  Low CPU on both instances
```

**Conclusion**: Railway network limits. Next steps:
- Deploy 2-3 instances behind load balancer
- Each instance gets separate network allocation

## Success Metrics

After running this test, you'll know:

1. **Sustained RPS Capacity**: The highest RPS you can maintain with <5% error rate
2. **Primary Bottleneck**: CPU, memory, network, or database connections
3. **Scaling Strategy**: What to optimize next

## Cost-Benefit of Scaling

Based on test results, here are typical next steps:

| Bottleneck | Solution | Cost | Time | Expected Gain |
|-----------|----------|------|------|---------------|
| DB Connections | Increase pool size | $0 | 1 day | 2-3x RPS |
| DB CPU | Read replicas | $50/mo | 1 week | 2x reads |
| Network | Horizontal scaling (3 instances) | $60/mo | 2 days | 3x throughput |
| App CPU | Horizontal scaling | $20/mo/instance | 2 days | Linear |

## Optional: Find True Limit

If extreme-stress-test.js succeeds, you can push even further:

```bash
# Test up to 20k VUs (requires more k6 client resources)
k6 run find-limit-test.js
```

## Questions to Answer

After the test, document:
- [ ] What RPS did you achieve at peak load?
- [ ] What was the error rate at peak?
- [ ] Which Railway metric maxed out first?
- [ ] At what VU count did performance degrade?
- [ ] What's your next scaling bottleneck?

---

**Ready to find your limit? Let's go!**

```bash
k6 run extreme-stress-test.js
```
