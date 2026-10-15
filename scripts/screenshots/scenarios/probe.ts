// The throwaway OAuth app of the repository (cmd/oauth-probe, what `go tool mage probe:oauth`
// runs), started against a scenario's server. It stands in for Claude web: it registers itself,
// prints the address of the consent page and waits for the browser to come back to its loopback
// port. It is a binary of its own, built before the scenarios run (see binary()).
import { spawn, spawnSync, type ChildProcess } from 'node:child_process'
import { existsSync } from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import type { Server } from './server.ts'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..', '..')
const exe = process.platform === 'win32' ? '.exe' : ''

let built: string | undefined
/**
 * The probe's binary: OAUTH_PROBE_BIN, else the one of the container image, else one built from
 * the sources at the first call (synchronously, so parallel scenarios do not build it twice).
 */
export function binary(): string {
    const wanted = process.env.OAUTH_PROBE_BIN
    if (wanted) return wanted
    const image = '/usr/local/bin/oauth-probe'
    if (existsSync(image)) return image
    if (!built) {
        const out = path.join(os.tmpdir(), 'savvy-scenarios', `oauth-probe${exe}`)
        console.log(`scenarios: no OAUTH_PROBE_BIN; building ${out}`)
        const res = spawnSync('go', ['build', '-o', out, './cmd/oauth-probe'], { cwd: root, stdio: 'inherit' })
        if (res.status !== 0) throw new Error('go build failed; set OAUTH_PROBE_BIN to a built oauth-probe')
        built = out
    }
    return built
}

export interface OauthProbe {
    /** The consent page address the probe asks to be opened; it appears once the app has registered. */
    authorizeUrl: Promise<string>
    /** Resolves with everything the probe printed once all its steps passed; rejects with it otherwise. */
    finished: Promise<string>
    stop(): void
}

export function startOauthProbe(server: Server): OauthProbe {
    const child: ChildProcess = spawn(binary(), ['-server', server.url, '-no-browser'], { stdio: ['ignore', 'pipe', 'pipe'] })
    let output = ''
    let found: (url: string) => void
    let lost: (e: Error) => void
    const authorizeUrl = new Promise<string>((resolve, reject) => {
        found = resolve
        lost = reject
    })
    const timer = setTimeout(() => lost(new Error(`the probe printed no address in 30 s:\n${output}`)), 30_000)
    const read = (chunk: Buffer) => {
        output += chunk.toString()
        const m = output.match(/^\s+(http\S+\/oauth\/authorize\?\S+)\s*$/m)
        if (m) found(m[1])
    }
    child.stdout?.on('data', read)
    child.stderr?.on('data', read)

    const finished = new Promise<string>((resolve, reject) => {
        child.once('error', reject)
        child.once('exit', (code) => {
            clearTimeout(timer)
            if (code === 0 && output.includes('All steps passed.')) resolve(output)
            else {
                const e = new Error(`the probe ended with ${code}:\n${output}`)
                lost(e)
                reject(e)
            }
        })
    })
    // A rejection nobody awaits yet (the scenario is still in the browser) must not end the run.
    authorizeUrl.catch(() => {})
    finished.catch(() => {})
    return { authorizeUrl, finished, stop: () => void child.kill() }
}
