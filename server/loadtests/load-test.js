import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter, Rate } from 'k6/metrics';
import { BASE_URL, thresholds, generateRandomFacts } from './config.js';

// Custom metrics to track business-specific performance
const evaluationTime = new Trend('evaluation_time_ms');  // From API response
const rulesMatched = new Counter('rules_matched_total');
const evaluationSuccess = new Rate('evaluation_success');

// Load test configuration
export const options = {
  stages: [
    { duration: '30s', target: 10 },   // Warm-up: 0 → 10 VUs
    { duration: '1m', target: 50 },    // Ramp-up: 10 → 50 VUs
    { duration: '3m', target: 50 },    // Sustained load: 50 VUs
    { duration: '1m', target: 100 },   // Peak test: 50 → 100 VUs
    { duration: '3m', target: 100 },   // Peak hold: 100 VUs
    { duration: '1m', target: 0 },     // Cool-down: 100 → 0 VUs
  ],
  thresholds: thresholds,
};

// Setup function - runs once before all VUs start
// Discovers existing tenants from the database (created by seed.js)
export function setup() {
  console.log('Discovering tenants from database...');

  // Fetch all tenants
  const tenantsRes = http.get(`${BASE_URL}/api/v1/tenants`);

  if (tenantsRes.status !== 200) {
    throw new Error(`Failed to fetch tenants: ${tenantsRes.status} ${tenantsRes.body}`);
  }

  const tenants = JSON.parse(tenantsRes.body);

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

    const rules = JSON.parse(rulesRes.body);
    const ruleIds = rules.map(r => r.id);

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

  console.log(`Setup complete! Ready to test with ${tenantData.length} tenants`);

  return {
    tenants: tenantData
  };
}

// Main test function - runs for each VU iteration
export default function(data) {
  // Randomly select a tenant for this iteration (simulates multi-tenant load)
  const tenant = data.tenants[Math.floor(Math.random() * data.tenants.length)];

  // Request distribution: 80% evaluate, 15% get rules, 5% health
  const rand = Math.random();

  if (rand < 0.80) {
    // 80% - Rule evaluation (core business logic)
    evaluateRules(tenant);
  } else if (rand < 0.95) {
    // 15% - Get rules (realistic read pattern)
    getRules(tenant);
  } else {
    // 5% - Health check
    healthCheck();
  }

  sleep(0.1); // Small delay between iterations
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

  // Track custom metrics from response
  if (res.status === 200) {
    try {
      const body = JSON.parse(res.body);

      // Track evaluation time from API response
      if (body.evaluationTime) {
        evaluationTime.add(body.evaluationTime);
      }

      // Count how many rules matched
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
        return Array.isArray(body);
      } catch (e) {
        return false;
      }
    },
  });
}

// Health check endpoint
function healthCheck() {
  const res = http.get(
    `${BASE_URL}/api/v1/health`,
    {
      tags: { endpoint: 'health' }
    }
  );

  check(res, {
    'health: status is 200': (r) => r.status === 200,
  });
}
