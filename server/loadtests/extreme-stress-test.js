import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter, Rate } from 'k6/metrics';
import { BASE_URL, generateRandomFacts } from './config.js';

// Custom metrics
const evaluationTime = new Trend('evaluation_time_ms');
const rulesMatched = new Counter('rules_matched_total');
const evaluationSuccess = new Rate('evaluation_success');

// EXTREME stress test - find the breaking point!
export const options = {
  stages: [
    { duration: '30s', target: 100 },   // Quick ramp
    { duration: '30s', target: 300 },   // Push higher
    { duration: '1m', target: 500 },    // 500 VUs!
    { duration: '2m', target: 500 },    // Hold at 500
    { duration: '30s', target: 750 },   // Push to 750
    { duration: '1m', target: 750 },    // Hold at 750
    { duration: '30s', target: 1000 },  // 1000 VUs - find the limit!
    { duration: '1m', target: 1000 },   // Hold at max
    { duration: '30s', target: 0 },     // Cool down
  ],
  // Very relaxed thresholds - we expect to break things
  thresholds: {
    'http_req_duration': ['p(95)<1000', 'p(99)<2000'],  // Allow high latency
    'http_req_failed': ['rate<0.10'],                    // Allow up to 10% errors
    'checks': ['rate>0.85'],                             // 85% success acceptable
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
  console.log('Target: 1000 concurrent VUs - Let\'s find the breaking point!');

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
