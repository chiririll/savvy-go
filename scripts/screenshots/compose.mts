// scripts/package.json "screenshots" (npm run screenshots in the root): runs
// docker-compose.screenshots.yml and removes what it created afterwards, whether or not
// the run worked. Needs only Node and Docker; the daemon may be remote (docker context): nothing is
// bind-mounted, the files are copied out of the container.
//
//   npm run screenshots              the default config: everything, in English and the light theme
//   npm run screenshots -- <name>    the config configs/<name>.json
//   npm run screenshots -- <name> --languages=en,ru --detail=key    settings on top of it
//   npm run screenshots -- --concurrency=8    browser contexts at a time (default: the CPUs of the Docker host)
import { spawn, type ChildProcess } from 'node:child_process'
import { existsSync, mkdirSync, readdirSync, readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..', '..')

// The config is a file of configs/, which goes into the image with the script, so only its name is passed on.
const configsDir = path.join(here, 'configs')
const names = existsSync(configsDir) ? readdirSync(configsDir).filter((f) => f.endsWith('.json')).map((f) => f.slice(0, -5)).sort() : []
// `--key=value` settings (see docs/scripts.md) go to the container as they are; config.ts checks them.
const args = process.argv.slice(2)
const overrides: Record<string, string> = {}
for (const arg of args.filter((a) => a.startsWith('--'))) {
    const [key, ...rest] = arg.slice(2).split('=')
    overrides[key] = rest.length ? rest.join('=') : 'true'
}
// Not a setting of the run's config: the runner reads it from the environment.
const concurrency = overrides.concurrency ?? ''
delete overrides.concurrency
if (concurrency && !/^[1-9]\d*$/.test(concurrency)) {
    console.error(`--concurrency must be a whole number above 0, not "${concurrency}".`)
    process.exit(2)
}
const config = (args.find((a) => !a.startsWith('--')) ?? '').replace(/\.json$/, '')
if (config && !names.includes(config)) {
    console.error(`No config "${config}"; there are ${names.join(', ') || 'none'} in scripts/screenshots/configs.`)
    process.exit(2)
}

// The directory of this run in screenshots/, named by the start time here, on the host, since the
// container knows only UTC (and by the config). RUN_NAME can be set by hand to name it otherwise.
const pad = (n: number) => String(n).padStart(2, '0')
const now = new Date()
const stamp = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}_${pad(now.getHours())}-${pad(now.getMinutes())}-${pad(now.getSeconds())}`
const runName = process.env.RUN_NAME || (config ? `${stamp}_${config}` : stamp)

// The app's config files go to the container as variables: a mount would name a path on the daemon's host.
const appConfig = readFileSync(path.join(here, 'config.toml'), 'utf8')
const appSeed = readFileSync(path.join(here, 'seed.toml'), 'utf8')

let current: ChildProcess | null = null
const compose = (...args: string[]) =>
    new Promise<number>((resolve) => {
        const child = spawn('docker', ['compose', '-f', 'docker-compose.screenshots.yml', ...args], {
            cwd: root,
            stdio: 'inherit',
            env: { ...process.env, APP_CONFIG: appConfig, APP_SEED: appSeed, RUN_NAME: runName, CONFIG: config, CONCURRENCY: concurrency, CONFIG_OVERRIDES: JSON.stringify(overrides) },
        })
        current = child
        child.on('error', (e) => {
            console.error(e.message)
            resolve(1)
        })
        child.on('close', (code) => resolve(code ?? 1))
    })

// Ctrl+C reaches docker compose as well (the terminal sends it to the whole group), which then stops
// the containers by itself. This script must not die with it: it has to go on to remove them. A second
// Ctrl+C is left to do what it does.
let interrupted = false
for (const signal of ['SIGINT', 'SIGTERM']) {
    process.once(signal, () => {
        interrupted = true
        // From a terminal the child has the signal already; from anywhere else it has not.
        current?.kill('SIGINT')
        console.log(`
${signal}: stopping and removing the containers...`)
    })
}

const code = await compose('up', '--build', '--abort-on-container-exit', '--exit-code-from', 'shots')
// Before `down`, which removes the container: it is stopped by now, but cp still reads it. Also after a
// failed run, which leaves FAILED.png.
mkdirSync(path.join(root, 'screenshots'), { recursive: true })
if ((await compose('cp', `shots:/out/${runName}`, 'screenshots')) !== 0) console.error('Could not copy the screenshots out of the container.')
await compose('down', '-v', '--remove-orphans')
if (interrupted) {
    console.log('Interrupted.')
    process.exit(130)
}
console.log(`Screenshots: ${path.join(root, 'screenshots', runName)}`)
process.exit(code)
