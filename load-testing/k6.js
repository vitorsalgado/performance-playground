import http from 'k6/http'
import { check, sleep } from 'k6'
import exec from 'k6/execution'

// Target
const BASE_URL = __ENV.BASE_URL || 'http://localhost:9999'
const AD_PATH = __ENV.AD_PATH || '/ad'

// Load profile
const VUS = Number(__ENV.VUS) || 50
const DURATION = __ENV.DURATION || '10m'
const SLEEP_SECONDS = Number(__ENV.SLEEP_SECONDS) || 0.1

// Randomization bounds (inclusive); lower bound applies to both app and publisher.
// Defaults align with gen-apps (start-id 1250, publisher-count 500, count 500000).
const MIN_ID = Number(__ENV.MIN_ID) || 1250
const MAX_PUBLISHER_ID = Number(__ENV.MAX_PUBLISHER_ID) || 1749   // 1250 + 500 - 1
const MAX_APP_ID = Number(__ENV.MAX_APP_ID) || 501249             // 1250 + 500000 - 1

export const options = {
  vus: VUS,
  duration: DURATION,
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<2000', 'p(99)<2000'],
  },
  tags: {
    test: 'exchange-ad',
  },
}

function randIntInclusive(min, max) {
  const lo = Number(min)
  const hi = Number(max)
  if (!Number.isFinite(lo) || !Number.isFinite(hi) || hi < lo) return lo
  return Math.floor(Math.random() * (hi - lo + 1)) + lo
}

function makeBidRequest({ appId, publisherId }) {
  const requestID = `k6-${exec.vu.idInTest}-${__ITER}-${Date.now()}`

  // Mirrors `internal/openrtb/openrtb.go` JSON tags (OpenRTB 2.1-ish).
  return {
    id: requestID,
    imp: [
      {
        id: '1',
        banner: { w: 200, h: 200 },
        bidfloor: 0.01,
        bidfloorcur: 'USD',
        secure: 0,
      },
    ],
    app: {
      id: String(appId), // exchange expects numeric string (strconv.Atoi)
      name: 'k6-load-test',
      domain: 'example.com',
      bundle: 'com.example.k6',
      publisher: {
        id: String(publisherId),
        name: 'k6-publisher',
      },
      privacypolicy: 0,
      paid: 0,
    },
    device: {
      ua: 'k6',
      ip: '127.0.0.1',
      os: 'linux',
      osv: 'unknown',
      language: 'en',
    },
    user: {
      id: `u-${exec.vu.idInTest}`,
    },
    at: 1,
    tmax: 500,
    test: 0,
    cur: ['USD'],
  }
}

export default function () {
  const publisherId = randIntInclusive(MIN_ID, MAX_PUBLISHER_ID)
  const appId = randIntInclusive(MIN_ID, MAX_APP_ID)

  const url = `${BASE_URL}${AD_PATH}`
  const payload = JSON.stringify(makeBidRequest({ appId, publisherId }))

  const res = http.post(url, payload, {
    // k6 will gzip the body when compression is set.
    compression: 'gzip',
    headers: {
      'Content-Type': 'application/json',
      'Content-Encoding': 'gzip',
    },
    tags: { endpoint: 'ad' },
    timeout: '2s',
  })

  check(res, {
    '200 (OK)': r => r.status === 200,
  })

  sleep(SLEEP_SECONDS)
}
