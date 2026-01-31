import http from 'k6/http'
import { check, sleep } from 'k6'

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'
const PING_PATH = __ENV.PING_PATH || '/ping'

const vus = Number(__ENV.VUS) || 20
const duration = __ENV.DURATION || '30s'

export const options = {
  vus,
  duration,
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<200', 'p(99)<500'],
  },
  tags: {
    test: 'exchange-ping',
  },
}

export default function () {
  const url = `${BASE_URL}${PING_PATH}`
  const res = http.get(url, {
    tags: { endpoint: 'ping' },
  })

  check(res, {
    '200 (OK)': r => r.status === 200,
    'pong': r => r.body === 'pong',
  })

  sleep(0.1)
}
