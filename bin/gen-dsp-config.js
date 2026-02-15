#!/usr/bin/env node
'use strict'

/**
 * Generates d/dsps.json (exchange DSP list with id, name, endpoint, latency) and
 * d/dsp-latencies.json (hostname -> latency for DSP replicas). Reads DSP_COUNT from
 * .env or --count. Latencies cycle: 0, 5ms, 10ms, 1s, 500ms.
 * No third-party dependencies.
 */

const fs = require('node:fs')
const path = require('node:path')

const LATENCY_CYCLE = ['0', '5ms', '10ms', '1s', '500ms']
const DEFAULT_COUNT = 25
const PROJECT_NAME = 'adtech'
const DSP_SERVICE = 'dsp'
const DSP_PORT = 8080
const BID_PATH = '/bid'

function loadEnv(envPath) {
  const out = {}
  try {
    const raw = fs.readFileSync(envPath, 'utf8')
    for (const line of raw.split('\n')) {
      const trimmed = line.trim()
      if (!trimmed || trimmed.startsWith('#')) continue
      const idx = trimmed.indexOf('=')
      if (idx === -1) continue
      const key = trimmed.slice(0, idx).trim()
      const val = trimmed.slice(idx + 1).trim()
      if (key) out[key] = val
    }
  } catch (e) {
    if (e.code !== 'ENOENT') throw e
  }
  return out
}

function usage(exitCode = 0) {
  const msg = `
Usage:
  node bin/gen-dsp-config.js [--count <N>] [--out-dsps <path>] [--out-latencies <path>] [--env <path>]

Options:
  --count         Number of DSPs (default: from .env DSP_COUNT or ${DEFAULT_COUNT})
  --out-dsps      Output path for dsps.json (default: d/dsps.json)
  --out-latencies Output path for dsp-latencies.json (default: d/dsp-latencies.json)
  --env           Path to .env file (default: .env in cwd)
  --help          Show this help

Examples:
  node bin/gen-dsp-config.js
  node bin/gen-dsp-config.js --count 10 --out-dsps d/dsps.json --out-latencies d/dsp-latencies.json
`
  process.stderr.write(msg.trim() + '\n')
  process.exit(exitCode)
}

function parseArgs(argv) {
  const cwd = process.cwd()
  const args = {
    count: null,
    outDsps: path.join(cwd, 'd', 'dsps.json'),
    outLatencies: path.join(cwd, 'd', 'dsp-latencies.json'),
    envPath: path.join(cwd, '.env'),
  }
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i]
    if (a === '--help' || a === '-h') usage(0)
    if (a.startsWith('--') && argv[i + 1] !== undefined) {
      const key = a.slice(2)
      const value = argv[++i]
      if (key === 'count') args.count = parseInt(value, 10)
      else if (key === 'out-dsps') args.outDsps = path.isAbsolute(value) ? value : path.join(cwd, value)
      else if (key === 'out-latencies') args.outLatencies = path.isAbsolute(value) ? value : path.join(cwd, value)
      else if (key === 'env') args.envPath = path.isAbsolute(value) ? value : path.join(cwd, value)
    }
  }
  return args
}

function main() {
  const argv = process.argv.slice(2)
  const args = parseArgs(argv)

  const env = loadEnv(args.envPath)
  const count = args.count != null ? args.count : (parseInt(env.DSP_COUNT, 10) || DEFAULT_COUNT)
  if (!Number.isInteger(count) || count < 1) {
    process.stderr.write(`gen-dsp-config: invalid count: ${count}\n`)
    process.exit(1)
  }

  const dsps = []
  const latencies = {}
  for (let i = 1; i <= count; i++) {
    const hostname = `${PROJECT_NAME}_${DSP_SERVICE}_${i}`
    const latency = LATENCY_CYCLE[(i - 1) % LATENCY_CYCLE.length]
    dsps.push({
      id: 1000 + i,
      name: `dsp${i}`,
      endpoint: `https://${hostname}:${DSP_PORT}${BID_PATH}`,
      latency,
    })
    latencies[hostname] = latency
  }

  const dspsDir = path.dirname(args.outDsps)
  const latenciesDir = path.dirname(args.outLatencies)
  if (!fs.existsSync(dspsDir)) fs.mkdirSync(dspsDir, { recursive: true })
  if (!fs.existsSync(latenciesDir)) fs.mkdirSync(latenciesDir, { recursive: true })

  fs.writeFileSync(args.outDsps, JSON.stringify(dsps, null, 2) + '\n', 'utf8')
  fs.writeFileSync(args.outLatencies, JSON.stringify(latencies, null, 2) + '\n', 'utf8')

  process.stdout.write(`gen-dsp-config: wrote ${count} DSPs to ${args.outDsps} and ${args.outLatencies}\n`)
}

main()
