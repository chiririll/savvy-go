// `npm run screenshots -- --local`: what docker-compose.screenshots.yml does, on this machine. Builds the
// frontend with the test ids (APP_ENV=screenshots) and the app into a temporary directory, so public/build
// and dist/ stay as they are, seeds the app on empty data, runs run.ts against it, then removes it all.
// Needs Go, and the browser of scripts/: `npx --prefix scripts playwright install chromium`.
import { spawn, type ChildProcess } from 'node:child_process'
import { cpSync, existsSync, mkdirSync, mkdtempSync, openSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { createServer } from 'node:net'
import os from 'node:os'
import path from 'node:path'
import { onCleanup, run, wasInterrupted } from '../runner.mts'
import { localAppConfig, localSeedConfig } from './args.mts'
import type { Run } from './main.mts'

const exe = process.platform === 'win32' ? '.exe' : ''

const freePort = () =>
    new Promise<number>((resolve, reject) => {
        const probe = createServer()
        probe.once('error', reject)
        probe.listen(0, '127.0.0.1', () => {
            const { port } = probe.address() as { port: number }
            probe.close(() => resolve(port))
        })
    })

export async function runLocally({ root, here, env }: Run): Promise<number> {
    if (!existsSync(path.join(here, '..', 'node_modules', 'playwright'))) {
        console.error('scripts/ has no dependencies: run `npm --prefix scripts ci` and `npx --prefix scripts playwright install chromium`.')
        return 2
    }
    const tmp = mkdtempSync(path.join(os.tmpdir(), 'savvy-screenshots-'))
    let app: ChildProcess | null = null
    onCleanup(async () => {
        if (app && app.exitCode === null) {
            app.kill()
            await new Promise<void>((resolve) => app!.once('exit', () => resolve()))
        }
        // The database files can stay locked for a moment after the process is gone (Windows).
        rmSync(tmp, { recursive: true, force: true, maxRetries: 10, retryDelay: 100 })
    })

    // A binary without the embedded frontend serves public/ of its working directory: this one.
    const web = path.join(tmp, 'web')
    cpSync(path.join(root, 'public'), path.join(web, 'public'), { recursive: true, filter: (src) => path.relative(root, src) !== path.join('public', 'build') })
    const build = { ...process.env, APP_ENV: 'screenshots', APP_VERSION: 'screenshots' }
    console.log('screenshots: building the frontend with test ids')
    if ((await run('npx', ['vite', 'build', '--outDir', path.join(web, 'public', 'build'), '--emptyOutDir'], { cwd: root, env: build })) !== 0) return 1
    if (wasInterrupted()) return 130
    const bin = path.join(tmp, `savvy-go${exe}`)
    console.log('screenshots: building the app')
    const ldflags = '-X main.Version=screenshots -X main.Env=screenshots'
    if ((await run('go', ['build', '-ldflags', ldflags, '-o', bin, '.'], { cwd: root, env: { ...process.env, CGO_ENABLED: '0' } })) !== 0) return 1
    if (wasInterrupted()) return 130

    const port = await freePort()
    const url = `http://127.0.0.1:${port}`
    const dataDir = path.join(tmp, 'data')
    mkdirSync(dataDir, { recursive: true })
    const manifest = path.join(dataDir, 'manifest.json')
    const configFile = path.join(tmp, 'config.toml')
    const seedFile = path.join(tmp, 'seed.toml')
    writeFileSync(configFile, localAppConfig(readFileSync(path.join(here, 'config.toml'), 'utf8'), { url, listen: `127.0.0.1:${port}`, dataDir }))
    writeFileSync(seedFile, localSeedConfig(readFileSync(path.join(here, 'seed.toml'), 'utf8'), manifest))

    const logFile = path.join(tmp, 'app.log')
    const log = () => (existsSync(logFile) ? readFileSync(logFile, 'utf8') : '')
    const fd = openSync(logFile, 'w')
    // Its own process group, so Ctrl+C in the terminal does not stop it before the cleanup does.
    app = spawn(bin, ['--seed-config', seedFile], { cwd: web, env: { ...process.env, CONFIG_FILE: configFile, TZ: 'UTC' }, stdio: ['ignore', fd, fd], detached: true })
    console.log(`screenshots: seeding and starting the app at ${url} (log: ${logFile})`)
    for (let i = 0; ; i++) {
        if (wasInterrupted()) return 130
        if (app.exitCode !== null) {
            console.error(`The app exited with ${app.exitCode} at start:\n${log()}`)
            return 1
        }
        try {
            if ((await fetch(`${url}/livez`)).ok && existsSync(manifest)) break
        } catch {
            // not listening yet
        }
        // As long as the compose healthcheck waits: 60 s start, then 20 tries of 3 s.
        if (i > 600) {
            console.error(`The app did not answer in 2 minutes:\n${log()}`)
            return 1
        }
        await new Promise((r) => setTimeout(r, 200))
    }

    return run('npx', ['tsx', 'screenshots/run.ts'], {
        cwd: path.join(here, '..'),
        env: {
            ...process.env,
            ...env,
            BASE_URL: url,
            MANIFEST: manifest,
            OUT: path.join(root, 'screenshots'),
            TZ: 'UTC',
            // The scenarios start the same binary, on the same test-id frontend.
            SAVVY_BIN: bin,
            SAVVY_WEB_DIR: web,
        },
    })
}
