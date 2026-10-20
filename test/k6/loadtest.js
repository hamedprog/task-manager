import http from 'k6/http';
import { check } from 'k6';

const baseURL = (__ENV.BASE_URL || 'http://localhost:9090').replace(/\/+$/, '');

export const options = {
  stages: [
    { duration: '10s', target: 5 },
    { duration: '30s', target: 5 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    checks: ['rate>0.99'],
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  const response = http.get(`${baseURL}/tasks?limit=50`);

  check(response, {
    'GET /tasks returns 200': (res) => res.status === 200,
    'GET /tasks returns a JSON array': (res) => {
      if (res.status !== 200) {
        return false;
      }

      try {
        return Array.isArray(res.json());
      } catch {
        return false;
      }
    },
  });
}
