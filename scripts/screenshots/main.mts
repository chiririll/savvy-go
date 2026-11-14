// scripts/package.json "screenshots" (npm run screenshots in the root; docs/scripts.md): every page of a
// throwaway app with demo data, then the scenarios. In Docker (the default, docker.mts) or here
// (--local, local.mts: needs Go and `npx --prefix scripts playwright install chromium`).
//
//   npm run screenshots              the default config: everything, in English and the light theme
//   npm run screenshots -- <name>    the config configs/<name>.json
//   npm run screenshots -- <name> --languages=en,ru --detail=key    settings on top of it
//   npm run screenshots -- --concurrency=8    browser contexts at a time (default: the CPUs where it runs)
//   npm run screenshots -- --local full       on this machine
import { existsSync, readdirSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { finish, modeOrExit, trapInterrupts } from '../runner.mts'
import { parseArgs, runName as defaultRunName } from './args.mts'
import { runInDocker } from './docker.mts'
import { runLocally } from './local.mts'

export interface Run {
    root: string
    here: string
    runName: string
    /** The environment the runner (run.ts) reads: CONFIG, CONCURRENCY, CONFIG_OVERRIDES, RUN_NAME. */
    env: Record<string, string>
}

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..', '..')

const { mode, rest } = modeOrExit()
// The config is a file of configs/, which goes into the image with the script, so only its name is passed on.
const configsDir = path.join(here, 'configs')
const names = existsSync(configsDir) ? readdirSync(configsDir).filter((f) => f.endsWith('.json')).map((f) => f.slice(0, -5)).sort() : []
let args
try {
    args = parseArgs(rest, names)
} catch (e) {
    console.error((e as Error).message)
    process.exit(2)
}

// Named by the start time here, on the host, since the container knows only UTC (and by the config).
// RUN_NAME can be set by hand to name it otherwise.
const runName = process.env.RUN_NAME || defaultRunName(new Date(), args.config)
const env = { RUN_NAME: runName, CONFIG: args.config, CONCURRENCY: args.concurrency, CONFIG_OVERRIDES: JSON.stringify(args.overrides) }

trapInterrupts()
const run: Run = { root, here, runName, env }
const code = mode === 'docker' ? await runInDocker(run) : await runLocally(run)
if (code === 0) console.log(`Screenshots: ${path.join(root, 'screenshots', runName)}`)
await finish(code)
