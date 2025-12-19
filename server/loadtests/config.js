// Shared k6 configuration for load tests
export const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Common thresholds for performance validation
export const thresholds = {
  'http_req_duration{p(95)}': ['p(95)<200'], // 95% of requests should complete under 200ms
  'http_req_duration{p(99)}': ['p(99)<500'], // 99% of requests should complete under 500ms
  'http_req_failed': ['rate<0.01'],           // Less than 1% error rate
  'checks': ['rate>0.95'],                    // 95% of custom checks should pass
};

// Helper function to generate random facts for evaluation
export function generateRandomFacts() {
  const countries = ['US', 'CA', 'UK', 'AU'];

  return {
    age: Math.floor(Math.random() * 60) + 18,                    // 18-77 years
    country: countries[Math.floor(Math.random() * countries.length)],
    is_premium: Math.random() > 0.7,                             // 30% premium users
    purchase_amount: Math.random() * 500                          // $0-500
  };
}

// Helper function to select random element from array
export function randomElement(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}
