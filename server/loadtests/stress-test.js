import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter, Rate } from 'k6/metrics';
import { BASE_URL, generateRandomFacts } from './config.js';

// Custom metrics
const evaluationTime = new Trend('evaluation_time_ms');
const rulesMatched = new Counter('rules_matched_total');
const evaluationSuccess = new Rate('evaluation_success');

// Aggressive stress test configuration - shorter but more intense
export const options = {
  stages: [
    { duration: '30s', target: 50 },    // Quick ramp-up
    { duration: '30s', target: 150 },   // Aggressive ramp to 150 VUs
    { duration: '2m', target: 150 },    // Hold at 150 VUs for 2 minutes
    { duration: '30s', target: 250 },   // Push to breaking point
    { duration: '1m', target: 250 },    // Hold at max for 1 minute
    { duration: '30s', target: 0 },     // Cool down
  ],
  // More relaxed thresholds for stress test (we expect to break some)
  thresholds: {
    'http_req_duration': ['p(95)<400', 'p(99)<800'],  // Higher latency acceptable
    'http_req_failed': ['rate<0.05'],                  // Allow up to 5% errors
    'checks': ['rate>0.90'],                           // 90% success acceptable
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

  // For each tenant, fetch their rules
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
    throw new Error('No valid tenants with rules found! Please run seed.js first.');
  }

  console.log(`Setup complete! Ready to stress test with ${tenantData.length} tenants`);

  return {
    tenants: tenantData
  };
}

// Main test function - more aggressive evaluation pattern
export default function(data) {
  const tenant = data.tenants[Math.floor(Math.random() * data.tenants.length)];

  // 90% evaluations, 10% reads - focused on core business logic
  const rand = Math.random();

  if (rand < 0.90) {
    evaluateRules(tenant);
  } else {
    getRules(tenant);
  }

  sleep(0.05); // Shorter sleep = more aggressive load
}

// Evaluate rules with random facts
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
      // Ignore JSON parse errors
    }
  }
}

// Get all rules for a tenant
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
