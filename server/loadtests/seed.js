import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL } from './config.js';

// Seeding script to populate the database with tenants, schemas, and rules
// Run this once before load testing: k6 run seed.js

export const options = {
  // Run with a single VU for one iteration (just to seed data)
  vus: 1,
  iterations: 1,
};

// Sample rule expressions with varying complexity
const ruleTemplates = [
  // Simple rules
  { name: 'Age Check - Adult', expression: 'facts.age >= 18', description: 'User must be 18 or older' },
  { name: 'Age Check - Senior', expression: 'facts.age >= 65', description: 'User is a senior citizen' },
  { name: 'Premium User', expression: 'facts.is_premium == true', description: 'User has premium subscription' },
  { name: 'US User', expression: 'facts.country == "US"', description: 'User is from the United States' },
  { name: 'High Spender', expression: 'facts.purchase_amount > 100', description: 'Purchase amount exceeds $100' },

  // Moderate complexity
  { name: 'Premium US User', expression: 'facts.country == "US" && facts.is_premium', description: 'Premium user from US' },
  { name: 'Young Adult', expression: 'facts.age >= 18 && facts.age < 25', description: 'User between 18 and 25' },
  { name: 'High Value Non-Premium', expression: 'facts.purchase_amount > 200 && !facts.is_premium', description: 'High purchase without premium' },
  { name: 'International Premium', expression: 'facts.country != "US" && facts.is_premium', description: 'Premium user outside US' },
  { name: 'Mid-Range Purchase', expression: 'facts.purchase_amount >= 50 && facts.purchase_amount <= 200', description: 'Purchase between $50-$200' },

  // Complex rules
  { name: 'VIP Eligibility', expression: 'facts.age >= 21 && facts.is_premium && facts.purchase_amount > 100', description: 'Eligible for VIP status' },
  { name: 'North America High Value', expression: 'facts.country in ["US", "CA"] && facts.purchase_amount > 150', description: 'High value from North America' },
  { name: 'Senior Discount Eligible', expression: 'facts.age >= 65 && facts.country in ["US", "CA", "UK"] && facts.purchase_amount > 0', description: 'Senior discount eligibility' },
  { name: 'Premium Upgrade Target', expression: '!facts.is_premium && facts.purchase_amount > 300 && facts.age >= 25', description: 'Target for premium upgrade' },
  { name: 'International VIP', expression: 'facts.country in ["UK", "AU", "CA"] && facts.is_premium && facts.purchase_amount > 500 && facts.age >= 30', description: 'International VIP customer' },
];

export default function() {
  const numTenants = 5; // Number of tenants to create
  const rulesPerTenant = 10; // Number of rules per tenant

  console.log(`Starting database seeding...`);
  console.log(`Creating ${numTenants} tenants with ${rulesPerTenant} rules each`);
  console.log('---');

  const createdTenants = [];

  for (let i = 0; i < numTenants; i++) {
    const tenantName = `load-test-tenant-${i + 1}`;
    console.log(`\nCreating tenant: ${tenantName}`);

    // Create tenant
    const tenantPayload = JSON.stringify({ name: tenantName });
    const tenantRes = http.post(
      `${BASE_URL}/api/v1/tenants`,
      tenantPayload,
      { headers: { 'Content-Type': 'application/json' } }
    );

    const tenantCheck = check(tenantRes, {
      'tenant created': (r) => r.status === 201,
    });

    if (!tenantCheck) {
      console.error(`Failed to create tenant ${tenantName}: ${tenantRes.status} ${tenantRes.body}`);
      continue;
    }

    const tenant = JSON.parse(tenantRes.body);
    const tenantId = tenant.id;
    console.log(`✓ Created tenant: ${tenantId}`);

    // Create schema
    const schemaPayload = JSON.stringify({
      definition: {
        facts: {
          age: 'int',
          country: 'string',
          is_premium: 'bool',
          purchase_amount: 'float64'
        }
      }
    });

    const schemaRes = http.post(
      `${BASE_URL}/api/v1/tenants/${tenantId}/schema`,
      schemaPayload,
      { headers: { 'Content-Type': 'application/json' } }
    );

    const schemaCheck = check(schemaRes, {
      'schema created': (r) => r.status === 201,
    });

    if (!schemaCheck) {
      console.error(`Failed to create schema for ${tenantName}: ${schemaRes.status} ${schemaRes.body}`);
      continue;
    }

    console.log(`✓ Created schema`);

    // Create rules
    const createdRules = [];
    for (let j = 0; j < rulesPerTenant; j++) {
      const ruleTemplate = ruleTemplates[j % ruleTemplates.length];

      const rulePayload = JSON.stringify({
        name: `${ruleTemplate.name} (T${i + 1})`,
        expression: ruleTemplate.expression,
        description: ruleTemplate.description
      });

      const ruleRes = http.post(
        `${BASE_URL}/api/v1/tenants/${tenantId}/rules`,
        rulePayload,
        { headers: { 'Content-Type': 'application/json' } }
      );

      if (ruleRes.status === 201) {
        const rule = JSON.parse(ruleRes.body);
        createdRules.push(rule.id);
      } else {
        console.error(`Failed to create rule "${ruleTemplate.name}": ${ruleRes.status}`);
      }
    }

    console.log(`✓ Created ${createdRules.length} rules`);

    createdTenants.push({
      id: tenantId,
      name: tenantName,
      ruleCount: createdRules.length
    });
  }

  // Summary
  console.log('\n========================================');
  console.log('Database Seeding Complete!');
  console.log('========================================');
  console.log(`Total tenants created: ${createdTenants.length}`);
  console.log('\nTenant Summary:');

  createdTenants.forEach((t, idx) => {
    console.log(`  ${idx + 1}. ${t.name}`);
    console.log(`     ID: ${t.id}`);
    console.log(`     Rules: ${t.ruleCount}`);
  });

  console.log('\n✓ Ready for load testing!');
  console.log('Run: k6 run load-test.js');
  console.log('========================================\n');
}
