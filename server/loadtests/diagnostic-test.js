import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter, Rate, Gauge } from 'k6/metrics';
import { BASE_URL, generateRandomFacts } from './config.js';

// Network and connection metrics
const connectTime = new Trend('tcp_connect_time');
const tlsHandshake = new Trend('tls_handshake_time');
const waitingTime = new Trend('waiting_time'); // Time to first byte
const receivingTime = new Trend('receiving_time'); // Data transfer time
const dataReceived = new Counter('data_received_bytes');
const dataSent = new Counter('data_sent_bytes');
const activeConnections = new Gauge('active_connections');

// Diagnostic test - gradual ramp to find breaking point
export const options = {
  stages: [
    { duration: '30s', target: 500 },
    { duration: '1m', target: 1000 },
    { duration: '1m', target: 1500 },
    { duration: '1m', target: 2000 },
    { duration: '1m', target: 2500 },
    { duration: '1m', target: 3000 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    'http_req_duration': ['p(95)<1000', 'p(99)<2000'],
    'http_req_failed': ['rate<0.05'],
    'tcp_connect_time': ['p(95)<100'],  // TCP connection time
    'waiting_time': ['p(95)<500'],       // Server processing time
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

  console.log(`🔍 DIAGNOSTIC TEST - Ready with ${tenantData.length} tenants`);
  console.log('Finding the exact bottleneck with detailed metrics...');

  return {
    tenants: tenantData
  };
}

// Main test function with detailed timing
export default function(data) {
  const tenant = data.tenants[Math.floor(Math.random() * data.tenants.length)];
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

  // Capture detailed timing metrics
  if (res.timings) {
    // TCP connection time (how long to establish connection)
    connectTime.add(res.timings.connecting);

    // TLS handshake (if HTTPS)
    if (res.timings.tls_handshaking > 0) {
      tlsHandshake.add(res.timings.tls_handshaking);
    }

    // Waiting time (server processing - time to first byte)
    waitingTime.add(res.timings.waiting);

    // Receiving time (network transfer)
    receivingTime.add(res.timings.receiving);
  }

  // Track data transfer
  if (res.body) {
    dataReceived.add(res.body.length);
  }
  dataSent.add(payload.length);

  check(res, {
    'status is 200': (r) => r.status === 200,
    'has results': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.results && Array.isArray(body.results);
      } catch (e) {
        return false;
      }
    },
  });

  sleep(0.01);
}

// Custom summary to show network bottleneck indicators
export function handleSummary(data) {
  const summary = {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
  };

  // Add diagnostic insights
  console.log('\n=== BOTTLENECK ANALYSIS ===');

  const tcpConnect = data.metrics.tcp_connect_time;
  const waiting = data.metrics.waiting_time;
  const receiving = data.metrics.receiving_time;

  if (tcpConnect && tcpConnect.values['p(95)'] > 100) {
    console.log('⚠️  HIGH TCP CONNECTION TIME - Network/connection pool bottleneck likely');
  }

  if (waiting && waiting.values['p(95)'] > 500) {
    console.log('⚠️  HIGH WAITING TIME - Server processing bottleneck');
  }

  if (receiving && receiving.values['p(95)'] > 200) {
    console.log('⚠️  HIGH RECEIVING TIME - Network bandwidth saturation likely');
  }

  const dataRx = data.metrics.data_received_bytes?.values?.count || 0;
  const dataTx = data.metrics.data_sent_bytes?.values?.count || 0;
  const totalMB = (dataRx + dataTx) / (1024 * 1024);
  const testDuration = data.state.testRunDurationMs / 1000;
  const mbps = (totalMB * 8) / testDuration;

  console.log(`📊 Total data transferred: ${totalMB.toFixed(2)} MB`);
  console.log(`📊 Average throughput: ${mbps.toFixed(2)} Mbps`);

  if (mbps > 100) {
    console.log('⚠️  NETWORK BANDWIDTH - You may be hitting Railway network limits');
  }

  return summary;
}

function textSummary(data, options) {
  // Use k6's built-in summary
  return JSON.stringify(data, null, 2);
}
