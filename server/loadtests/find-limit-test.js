import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, generateRandomFacts } from './config.js';

// Aggressive test to find the TRUE breaking point after connection pool fix
export const options = {
  stages: [
    { duration: '1m', target: 2000 },
    { duration: '1m', target: 4000 },
    { duration: '1m', target: 6000 },
    { duration: '1m', target: 8000 },
    { duration: '1m', target: 10000 },  // Push to 10k VUs!
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    'http_req_duration': ['p(95)<1000'],
    'http_req_failed': ['rate<0.05'],
  },
};

export function setup() {
  const tenantsRes = http.get(`${BASE_URL}/api/v1/tenants`);
  const response = JSON.parse(tenantsRes.body);
  const tenants = response.tenants || [];

  const tenantData = [];
  for (const tenant of tenants) {
    const rulesRes = http.get(`${BASE_URL}/api/v1/tenants/${tenant.id}/rules`);
    const rulesResponse = JSON.parse(rulesRes.body);
    const rules = rulesResponse.rules || [];

    tenantData.push({
      id: tenant.id,
      name: tenant.name,
      ruleIds: rules.map(r => r.ID),
    });
  }

  return { tenants: tenantData };
}

export default function(data) {
  const tenant = data.tenants[Math.floor(Math.random() * data.tenants.length)];
  const facts = generateRandomFacts();

  const res = http.post(
    `${BASE_URL}/api/v1/evaluate`,
    JSON.stringify({ tenantId: tenant.id, facts: facts }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  check(res, {
    'status is 200': (r) => r.status === 200,
  });

  sleep(0.01);
}
