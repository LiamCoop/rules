# Load Testing with k6

This directory contains k6 load tests for the rule evaluation service.

## Prerequisites

1. **Install k6**:
   ```bash
   brew install k6
   ```

2. **Ensure the server is running**:
   ```bash
   # From the server directory
   DATABASE_URL="your-connection-string" go run ./cmd/server/main.go
   ```

   The server should be accessible at `http://localhost:8080`.

## Running the Load Test

### Step 1: Seed the Database (First Time Only)

Before running load tests, you need to populate the database with test data:

```bash
k6 run seed.js
```

This will create:
- **5 tenants** (load-test-tenant-1 through load-test-tenant-5)
- **Schema** for each tenant with a `facts` type containing:
  - `age` (int)
  - `country` (string)
  - `is_premium` (bool)
  - `purchase_amount` (float64)
- **10 rules** per tenant with varying complexity (simple, moderate, complex)

**You only need to run this once.** The data persists in your database and can be reused across multiple load test runs.

### Step 2: Run the Load Test

Run the comprehensive load test:

```bash
k6 run load-test.js
```

The load test will automatically discover all tenants created by the seed script and distribute load across them.

### Additional Options

Save results to file for later analysis:

```bash
k6 run --out json=results/results.json load-test.js
```

Test against a different URL:

```bash
BASE_URL=http://localhost:9000 k6 run load-test.js
```

### Resetting Test Data

To start fresh, you can delete the test tenants and re-run the seed script:

```bash
# Delete old test data (manual SQL or via API)
# Then re-seed:
k6 run seed.js
```

## What the Test Does

The load test simulates realistic multi-tenant traffic patterns:

1. **Setup Phase**: Discovers all tenants from the database (created by `seed.js`)

2. **Load Pattern** (~10 minutes total):
   - Warm-up: 0 → 10 virtual users over 30 seconds
   - Ramp-up: 10 → 50 VUs over 1 minute
   - Sustained: 50 VUs for 3 minutes
   - Peak: 50 → 100 VUs over 1 minute
   - Peak hold: 100 VUs for 3 minutes
   - Cool-down: 100 → 0 VUs over 1 minute

3. **Request Distribution**:
   - 80% rule evaluations (`POST /api/v1/evaluate`)
   - 15% rule retrievals (`GET /api/v1/tenants/{id}/rules`)
   - 5% health checks (`GET /api/v1/health`)

4. **Multi-Tenant Simulation**: Each request randomly selects one of the seeded tenants, simulating realistic multi-tenant load distribution

## Understanding the Results

After the test completes, k6 displays a summary with key metrics:

### Key Metrics to Watch

- **`http_reqs`**: Total number of requests made
- **`http_req_duration`**: Response time statistics
  - `p(95)`: 95% of requests completed under this time
  - `p(99)`: 99% of requests completed under this time
- **`http_req_failed`**: Percentage of failed requests
- **`iterations`**: Number of test iterations completed

### Custom Metrics

- **`evaluation_time_ms`**: Evaluation time reported by the API
- **`rules_matched_total`**: Total number of rule matches across all evaluations
- **`evaluation_success`**: Success rate of evaluation requests

### Performance Thresholds

The test enforces these thresholds:
- P95 latency < 200ms
- P99 latency < 500ms
- Error rate < 1%
- Check success rate > 95%

If any threshold fails, k6 will exit with a non-zero status code.

## Calculating RPS (Requests Per Second)

Look for the `http_reqs` metric in the output:

```
http_reqs......................: 45678 761.3/s
                                  ^^^^^  ^^^^^^
                                  total   RPS
```

The second number (in this example: 761.3/s) is your **average RPS** across the entire test.

For **peak RPS**, look at the metrics during the "Peak hold" stage when 100 VUs are active.

## Running Multiple Times for Consistency

For credible benchmarking, run the test 3 times and average the results:

```bash
k6 run load-test.js  # Run 1
k6 run load-test.js  # Run 2
k6 run load-test.js  # Run 3
```

**Note**: You don't need to re-seed between runs! The test data persists in your database, so you can run the load test as many times as you want.

Note the RPS from each run and calculate the average for your README.

## Troubleshooting

### "No tenants found!" Error

If the load test fails with "No tenants found! Please run seed.js first":
- Run the seed script first: `k6 run seed.js`
- Verify tenants were created: `curl http://localhost:8080/api/v1/tenants`

### Server Connection Errors

If you see connection errors:
- Verify the server is running: `curl http://localhost:8080/api/v1/health`
- Check the BASE_URL matches your server port
- Ensure your database is accessible

### High Error Rates

If error rate > 1%:
- Check server logs for errors
- Verify database connection pool isn't exhausted
- Reduce load (lower VU count in options.stages)

### Low RPS

If not hitting target RPS:
- Close other resource-intensive applications
- Increase PostgreSQL max_connections
- Run server in production/release mode
- Check for database query bottlenecks

## File Structure

- **`seed.js`**: Database seeding script (run once to create test data)
- **`load-test.js`**: Main load test script (run multiple times for benchmarking)
- **`config.js`**: Shared configuration and helper functions
- **`results/`**: Directory for saved test results (git-ignored)

## Next Steps

After establishing baseline performance:

1. Document results in the main README
2. Add more test scenarios (stress test, spike test)
3. Integrate into CI/CD for regression detection
4. Profile and optimize identified bottlenecks
