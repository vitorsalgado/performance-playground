#!/usr/bin/env node
'use strict'

/**
 * Generates a JSON array of `App` records as defined in `exchange/exchange.go`:
 *   type App struct {
 *     ID        int        `json:"id"`
 *     Name      string     `json:"name"`
 *     Publisher *Publisher `json:"publisher"`
 *   }
 *   type Publisher struct {
 *     ID   int    `json:"id"`
 *     Name string `json:"name"`
 *   }
 *
 * This script is streaming/backpressure-aware (memory efficient for massive outputs).
 * No third-party dependencies.
 */

const fs = require('node:fs')
const { once } = require('node:events')

function usage(exitCode = 0) {
  const msg = `
Usage:
  node bin/generate-apps-json.js --count <N> [--out <path|->] [--publisher-count <N>] [--start-id <N>]

Options:
  --count             Number of App records to generate (required, integer >= 0)
  --out               Output file path, or "-" for stdout (default: "-")
  --publisher-count   Number of distinct publishers to rotate through (default: 1000)
  --start-id          Starting App id (default: 1)
  --help              Show this help

Examples:
  node bin/generate-apps-json.js --count 1000 --out d/apps.json
  node bin/generate-apps-json.js --count 5000000 --out - > d/apps.json
`
  // eslint-disable-next-line no-console
  console.error(msg.trim())
  process.exit(exitCode)
}

function parseIntStrict(name, value) {
  if (value === undefined) return undefined
  if (!/^-?\d+$/.test(value)) throw new Error(`Invalid ${name}: ${value}`)
  const n = Number(value)
  if (!Number.isSafeInteger(n)) throw new Error(`Invalid ${name} (not a safe integer): ${value}`)
  return n
}

function parseArgs(argv) {
  const out = { outPath: '-', publisherCount: 1000, startId: 1 }

  for (let i = 0; i < argv.length; i++) {
    const a = argv[i]
    if (a === '--help' || a === '-h') usage(0)
    if (!a.startsWith('--')) throw new Error(`Unknown argument: ${a}`)

    const key = a.slice(2)
    const next = argv[i + 1]

    switch (key) {
      case 'count':
        out.count = parseIntStrict('count', next)
        i++
        break
      case 'out':
        if (next === undefined) throw new Error('Missing value for --out')
        out.outPath = next
        i++
        break
      case 'publisher-count':
        out.publisherCount = parseIntStrict('publisher-count', next)
        i++
        break
      case 'start-id':
        out.startId = parseIntStrict('start-id', next)
        i++
        break
      default:
        throw new Error(`Unknown option: --${key}`)
    }
  }

  if (out.count === undefined) throw new Error('Missing required --count')
  if (out.count < 0) throw new Error('--count must be >= 0')
  if (out.publisherCount === undefined || out.publisherCount <= 0) throw new Error('--publisher-count must be > 0')
  if (out.startId === undefined || out.startId < 0) throw new Error('--start-id must be >= 0')

  return out
}

function makeApp(appId, publisherCount) {
  const pubId = ((appId - 1) % publisherCount) + 1
  return {
    id: appId,
    name: `app-${appId}`,
    publisher: {
      id: pubId,
      name: `publisher-${pubId}`,
    },
  }
}

async function writeAll(stream, chunk) {
  if (!stream.write(chunk)) await once(stream, 'drain')
}

async function main() {
  let opts
  try {
    opts = parseArgs(process.argv.slice(2))
  } catch (err) {
    // eslint-disable-next-line no-console
    console.error(String(err && err.message ? err.message : err))
    usage(2)
    return
  }

  const outStream =
    opts.outPath === '-' ? process.stdout : fs.createWriteStream(opts.outPath, { encoding: 'utf8' })

  // Ensure we don't throw on EPIPE when piping to `head`, etc.
  outStream.on('error', (err) => {
    if (err && err.code === 'EPIPE') process.exit(0)
    // eslint-disable-next-line no-console
    console.error(err)
    process.exit(1)
  })

  const total = opts.count
  const startId = opts.startId

  await writeAll(outStream, '[')

  for (let i = 0; i < total; i++) {
    const appId = startId + i
    const app = makeApp(appId, opts.publisherCount)
    const json = JSON.stringify(app)

    if (i !== 0) await writeAll(outStream, ',')
    await writeAll(outStream, json)
  }

  await writeAll(outStream, ']')

  if (outStream !== process.stdout) {
    await new Promise((resolve) => outStream.end(resolve))
  }
}

main().catch((err) => {
  // eslint-disable-next-line no-console
  console.error(err)
  process.exit(1)
})
