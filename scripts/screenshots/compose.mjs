// scripts/package.json "screenshots" (npm run screenshots in the root): runs
// docker-compose.screenshots.yml and removes what it created afterwards, whether or not
// the run worked. Needs only Node and Docker.
//
//   npm run screenshots              the default config: everything, in English and the light theme
//   npm run screenshots -- <name>    the config configs/<name>.json
import { spawnSync } from 'node:child_process'
import { existsSync, readdirSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..', '..')

// The config is a file of configs/, which goes into the image with the script, so only its name is passed on.
const configsDir = path.join(here, 'configs')
const names = existsSync(configsDir) ? readdirSync(configsDir).filter((f) => f.endsWith('.json')).map((f) => f.slice(0, -5)).sort() : []
const config = (process.argv[2] ?? '').replace(/\.json$/, '')
if (config && !names.includes(config)) {
    console.error(`No config "${config}"; there are ${names.join(', ') || 'none'} in scripts/screenshots/configs.`)
    process.exit(2)
}

// The directory of this run in screenshots/, named by the start time here, on the host, since the
// container knows only UTC (and by the config). RUN_NAME can be set by hand to name it otherwise.
const pad = (n) => String(n).padStart(2, '0')
const now = new Date()
const stamp = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}_${pad(now.getHours())}-${pad(now.getMinutes())}-${pad(now.getSeconds())}`
const runName = process.env.RUN_NAME || (config ? `${stamp}_${config}` : stamp)

const compose = (...args) =>
    spawnSync('docker', ['compose', '-f', 'docker-compose.screenshots.yml', ...args], {
        cwd: root,
        stdio: 'inherit',
        env: { ...process.env, RUN_NAME: runName, CONFIG: config },
    })

const run = compose('up', '--build', '--abort-on-container-exit', '--exit-code-from', 'shots')
compose('down', '-v', '--remove-orphans')
console.log(`Screenshots: ${path.join(root, 'screenshots', runName)}`)
process.exit(run.status ?? 1)
