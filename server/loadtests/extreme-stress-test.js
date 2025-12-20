import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter, Rate } from 'k6/metrics';
import { BASE_URL, generateRandomFacts } from './config.js';

// Custom metrics
const evaluationTime = new Trend('evaluation_time_ms');
const rulesMatched = new Counter('rules_matched_total');
const evaluationSuccess = new Rate('evaluation_success');

// EXTREME stress test - find the REAL breaking point!
// Post connection-pool fix (MaxOpenConns=100, MaxIdleConns=50), we can handle much higher load
//
// Test Profile:
// - Total duration: ~11 minutes
// - Peak load: 15,000 concurrent VUs
// - Expected RPS at peak: 15,000+ RPS (based on 0.01s sleep)
//
// What to monitor during this test:
// 1. Railway Metrics:
//    - CPU usage (both app and DB instances)
//    - Memory consumption
//    - Network I/O (watch for saturation)
//    - Database connection count
// 2. k6 Output:
//    - When do error rates start increasing?
//    - At what VU count does P95 latency spike?
//    - http_req_failed rate (connection refused = hit limit)
//
// Expected bottlenecks (in order of likelihood):
// 1. Database connections (100 max configured)
// 2. CPU on database instance (CEL evaluation queries)
// 3. Network bandwidth (egress limits)
// 4. Application CPU (CEL evaluation overhead)
export const options = {
  stages: [
    // Warm-up phase
    { duration: '30s', target: 1000 },    // Quick ramp to baseline
    { duration: '30s', target: 1000 },    // Stabilize

    // Aggressive ramp to find limits
    { duration: '1m', target: 3000 },     // Known good from diagnostic test (6.9k RPS)
    { duration: '1m', target: 5000 },     // Push to 5k VUs
    { duration: '1m', target: 7500 },     // 7.5k VUs - likely still healthy
    { duration: '1m', target: 10000 },    // 10k VUs - serious load
    { duration: '1m', target: 12500 },    // 12.5k VUs - finding the edge
    { duration: '1m', target: 15000 },    // 15k VUs - MAXIMUM LOAD
    { duration: '2m', target: 15000 },    // Hold at peak to observe stability

    // Cool down
    { duration: '30s', target: 0 },
  ],
  // Relaxed thresholds - we're intentionally breaking things to find limits
  thresholds: {
    'http_req_duration': ['p(95)<2000', 'p(99)<5000'],  // Allow degradation under extreme load
    'http_req_failed': ['rate<0.15'],                    // Allow up to 15% errors at peak
    'checks': ['rate>0.80'],                             // 80% success acceptable when breaking
  },
};

// Setup function - discovers existing tenants
export function setup() {
  console.log('Discovering tenants from database...');

  const tenantsRes = http.get(`${BASE_URL}/api/v1/tenants`);

  if (tenantsRes.status !== 200) {
    throw new Error(`Failed to fetch tenants: ${tenantsRes.status} ${tenantsRes.body}`);
  }

  const response = JSON.parse(tenantsRes.body);
  const tenants = response.tenants || [];

  if (!Array.isArray(tenants) || tenants.length === 0) {
    throw new Error('No tenants found! Please run seed.js first: k6 run seed.js');
  }

  console.log(`Found ${tenants.length} tenants`);

  const tenantData = [];
  for (const tenant of tenants) {
    const rulesRes = http.get(`${BASE_URL}/api/v1/tenants/${tenant.id}/rules`);

    if (rulesRes.status !== 200) {
      console.warn(`Failed to fetch rules for tenant ${tenant.id}: ${rulesRes.status}`);
      continue;
    }

    const rulesResponse = JSON.parse(rulesRes.body);
    const rules = rulesResponse.rules || [];
    const ruleIds = rules.map(r => r.ID);

    tenantData.push({
      id: tenant.id,
      name: tenant.name,
      ruleIds: ruleIds,
      ruleCount: ruleIds.length
    });

    console.log(`✓ Tenant: ${tenant.name} (${ruleIds.length} rules)`);
  }

  if (tenantData.length === 0) {
    throw new Error('No valid tenants with rules found!');
  }

  console.log(`🚀 EXTREME STRESS TEST - Ready with ${tenantData.length} tenants`);
  console.log('🔥 Target: 15,000 concurrent VUs - Finding the REAL breaking point!');
  console.log('📊 Post connection-pool fix - expecting 10,000+ RPS sustained');
  console.log('⏱️  Test duration: ~11 minutes');
  console.log('');
  console.log('Watch Railway metrics during the test to see what breaks first!');

  return {
    tenants: tenantData
  };
}

// Main test function - 95% evaluations for maximum CPU stress
export default function(data) {
  const tenant = data.tenants[Math.floor(Math.random() * data.tenants.length)];

  // 95% evaluations - maximize CEL engine stress
  if (Math.random() < 0.95) {
    evaluateRules(tenant);
  } else {
    getRules(tenant);
  }

  sleep(0.01); // Very short sleep = maximum throughput
}

function evaluateRules(tenant) {
  const facts = generateRandomFacts();

  const payload = JSON.stringify({
    tenantId: tenant.id,
    facts: facts
  });

  const res = http.post(
    `${BASE_URL}/api/v1/evaluate`,
    payload,
    {
      headers: { 'Content-Type': 'application/json' },
      tags: { endpoint: 'evaluate' }
    }
  );

  const success = check(res, {
    'evaluate: status is 200': (r) => r.status === 200,
    'evaluate: has results': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.results && Array.isArray(body.results);
      } catch (e) {
        return false;
      }
    },
  });

  evaluationSuccess.add(success);

  if (res.status === 200) {
    try {
      const body = JSON.parse(res.body);

      if (body.evaluationTime) {
        const timeMs = parseFloat(body.evaluationTime.replace('ms', ''));
        evaluationTime.add(timeMs);
      }

      if (body.results) {
        const matchedCount = body.results.filter(r => r.matched).length;
        rulesMatched.add(matchedCount);
      }
    } catch (e) {
      // Ignore
    }
  }
}

function getRules(tenant) {
  const res = http.get(
    `${BASE_URL}/api/v1/tenants/${tenant.id}/rules`,
    {
      tags: { endpoint: 'get_rules' }
    }
  );

  check(res, {
    'get_rules: status is 200': (r) => r.status === 200,
    'get_rules: has rules array': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.rules && Array.isArray(body.rules);
      } catch (e) {
        return false;
      }
    },
  });
}
