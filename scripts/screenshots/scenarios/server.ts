// An app of its own for a scenario: the app's binary started on an empty data directory and a free
// port, thrown away afterwards. The binary is SAVVY_BIN, else the one of the container image, else
// one built from the sources (`go build`, once; it serves public/build from disk, so that must
// have been built with APP_ENV=screenshots for the marks the scenarios use).
import { spawn, spawnSync, type ChildProcess } from 'node:child_process'
import { existsSync, mkdirSync, openSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { createServer } from 'node:net'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..', '..')
const exe = process.platform === 'win32' ? '.exe' : ''

export interface Server {
    url: string
    dataDir: string
    /** The first-run password, read from the file the app keeps it in. */
    bootstrapPassword(): string
    /** Everything the app has logged so far, for a failure message. */
    log(): string
    stop(): Promise<void>
}

let built: string | undefined
/** The app's binary; built from the sources at the first call when there is none to use. */
export function binary(wanted: string): string {
    if (wanted) return wanted
    const image = '/usr/local/bin/savvy-go'
    if (existsSync(image)) return image
    if (!built) {
        const out = path.join(os.tmpdir(), 'savvy-scenarios', `savvy-go${exe}`)
        console.log(`scenarios: no SAVVY_BIN; building ${out}`)
        const res = spawnSync('go', ['build', '-o', out, '.'], { cwd: root, stdio: 'inherit' })
        if (res.status !== 0) throw new Error('go build failed; set SAVVY_BIN to a built binary')
        built = out
    }
    return built
}

const freePort = () =>
    new Promise<number>((resolve, reject) => {
        const probe = createServer()
        probe.once('error', reject)
        probe.listen(0, '127.0.0.1', () => {
            const { port } = probe.address() as { port: number }
            probe.close(() => resolve(port))
        })
    })

/**
 * Starts the app and waits until it answers. `label` names its directory: the same every run, so
 * a path shown on the screen (the bootstrap password file) does not change between runs.
 */
export async function startServer(bin: string, label: string): Promise<Server> {
    const base = path.join(os.tmpdir(), 'savvy-scenarios', label)
    rmSync(base, { recursive: true, force: true })
    const dataDir = path.join(base, 'data')
    mkdirSync(dataDir, { recursive: true })
    const port = await freePort()
    const url = `http://127.0.0.1:${port}`
    const configFile = path.join(base, 'config.toml')
    writeFileSync(configFile, `[server]\nlisten = "127.0.0.1:${port}"\ntimezone = "UTC"\n\n[paths]\ndata = ${JSON.stringify(dataDir)}\n`)

    const logFile = path.join(base, 'app.log')
    const fd = openSync(logFile, 'w')
    // cwd: a binary without the embedded frontend looks for public/ in it.
    const child: ChildProcess = spawn(bin, [], { cwd: root, env: { ...process.env, CONFIG_FILE: configFile }, stdio: ['ignore', fd, fd] })
    let exited: number | null = null
    child.once('exit', (code) => (exited = code ?? -1))
    const readLog = () => {
        try {
            return readFileSync(logFile, 'utf8')
        } catch {
            return ''
        }
    }
    const stop = async () => {
        if (exited === null) {
            child.kill()
            await new Promise<void>((resolve) => child.once('exit', () => resolve()))
        }
        // The database files can stay locked for a moment after the process is gone (Windows).
        rmSync(base, { recursive: true, force: true, maxRetries: 10, retryDelay: 100 })
    }

    for (let i = 0; ; i++) {
        if (exited !== null) throw new Error(`the app exited with ${exited} at start:\n${readLog()}`)
        try {
            if ((await fetch(`${url}/livez`)).ok) break
        } catch {
            // not listening yet
        }
        if (i > 150) {
            await stop()
            throw new Error(`the app did not answer in 30 s:\n${readLog()}`)
        }
        await new Promise((r) => setTimeout(r, 200))
    }
    return {
        url,
        dataDir,
        log: readLog,
        stop,
        bootstrapPassword: () => readFileSync(path.join(dataDir, 'bootstrap-password'), 'utf8').trim(),
    }
}
