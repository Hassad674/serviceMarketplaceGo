// k6-search.js — baseline load test for the /api/v1/search endpoint.
//
// Runs 50 virtual users for 30 seconds, each firing a mix of:
//   - empty listing      (weight 30%)
//   - text query         (weight 40%)
//   - filtered query     (weight 20%)
//   - text + filter mix  (weight 10%)
//
// SLOs enforced as k6 thresholds:
//   - http_req_failed rate                < 1%
//   - http_req_duration p(95)             < 200ms
//   - http_req_duration p(99)             < 500ms
//
// Usage:
//   # Local (default):
//   k6 run scripts/perf/k6-search.js
//
//   # Staging:
//   BASE_URL=https://staging.marketplace.example TOKEN=<bearer> \
//     k6 run scripts/perf/k6-search.js
//
// DO NOT RUN AGAINST PROD. The test floods the search endpoint with
// 50 VUs — running it against production would look identical to a
// denial-of-service. Staging is safe because it's isolated from real
// customer traffic.

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8083';
const TOKEN = __ENV.TOKEN || '';

// Custom metric: fraction of responses with found > 0. Useful to
// spot the "zero-result spike" symptom that indicates index drift.
const NonEmpty = new Rate('search_nonempty_fraction');

export const options = {
  stages: [
    { duration: '10s', target: 20 }, // ramp-up
    { duration: '20s', target: 50 }, // sustain
    { duration: '5s', target: 0 },   // ramp-down
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<200', 'p(99)<500'],
    search_nonempty_fraction: ['rate>0.7'],
  },
};

const TEXT_QUERIES = [
  'React', 'Node', 'Go', 'Python', 'TypeScript',
  'design', 'senior', 'paris', 'freelance', 'apporteur',
];

const COUNTRY_CODES = ['FR', 'DE', 'GB', 'NL', 'ES', 'PT', 'US'];

function pick(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

function randomScenario() {
  const dice = Math.random();
  if (dice < 0.3) {
    return { name: 'empty_listing', path: `/api/v1/search?persona=freelance&per_page=20` };
  }
  if (dice < 0.7) {
    return { name: 'text_query', path: `/api/v1/search?persona=freelance&q=${encodeURIComponent(pick(TEXT_QUERIES))}` };
  }
  if (dice < 0.9) {
    return { name: 'filter_query', path: `/api/v1/search?persona=freelance&country_code=${pick(COUNTRY_CODES)}` };
  }
  return {
    name: 'mixed_query',
    path: `/api/v1/search?persona=freelance&q=${encodeURIComponent(pick(TEXT_QUERIES))}&country_code=${pick(COUNTRY_CODES)}`,
  };
}

export default function () {
  const scenario = randomScenario();
  const headers = TOKEN ? { Authorization: `Bearer ${TOKEN}` } : {};
  const res = http.get(`${BASE_URL}${scenario.path}`, { headers, tags: { scenario: scenario.name } });

  check(res, {
    [`${scenario.name} status is 200`]: (r) => r.status === 200,
    [`${scenario.name} returned JSON`]: (r) => r.headers['Content-Type'] && r.headers['Content-Type'].indexOf('json') >= 0,
  });

  if (res.status === 200) {
    try {
      const body = JSON.parse(res.body);
      const found = (body.data && body.data.found) || 0;
      NonEmpty.add(found > 0);
    } catch (_err) {
      NonEmpty.add(false);
    }
  } else {
    NonEmpty.add(false);
  }

  sleep(Math.random() * 0.5);
}
