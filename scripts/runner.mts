// What the dev commands that can run in Docker or on this machine share (npm run stories, screenshots;
// docs/scripts.md): the --docker / --local flag, starting a child, and cleaning up after Ctrl+C.
//
// Docker mode calls plain `docker`, so it uses whatever context is active; nothing is bind-mounted, so
// any daemon works, local or not.
import { spawn, type ChildProcess, type SpawnOptions } from 'node:child_process'

export type Mode = 'docker' | 'local'

/**
 * The mode of a run and the arguments without the flag. `--docker` or `--local` wins, then
 * SAVVY_RUNNER (docker or local), then docker. What follows a bare `--` is not looked at.
 */
export function takeMode(args: string[], env: Record<string, string | undefined> = process.env): { mode: Mode; rest: string[] } {
    const end = args.indexOf('--')
    const head = end === -1 ? args : args.slice(0, end)
    const tail = end === -1 ? [] : args.slice(end)
    const flags = head.filter((a) => a === '--docker' || a === '--local')
    if (new Set(flags).size > 1) throw new Error('Pass either --docker or --local, not both.')
    const rest = [...head.filter((a) => a !== '--docker' && a !== '--local'), ...tail]
    if (flags.length) return { mode: flags[0].slice(2) as Mode, rest }

    const fromEnv = (env.SAVVY_RUNNER ?? '').trim().toLowerCase()
    if (fromEnv === '') return { mode: 'docker', rest }
    if (fromEnv === 'docker' || fromEnv === 'local') return { mode: fromEnv, rest }
    throw new Error(`SAVVY_RUNNER must be docker or local, not "${env.SAVVY_RUNNER}".`)
}

/** The mode of this process's arguments; a bad flag stops it with a message. */
export function modeOrExit(args = process.argv.slice(2)) {
    try {
        return takeMode(args)
    } catch (e) {
        console.error((e as Error).message)
        process.exit(2)
    }
}

let current: ChildProcess | null = null
let interrupted = false
const cleanups: (() => void | Promise<void>)[] = []

/** Runs a command with this terminal and resolves with its exit code (1 when it could not start). */
export function run(cmd: string, args: string[], opts: SpawnOptions = {}): Promise<number> {
    return new Promise((resolve) => {
        // npx and npm are .cmd files on Windows.
        const child = spawn(cmd, args, { stdio: 'inherit', shell: process.platform === 'win32', ...opts })
        current = child
        child.on('error', (e) => {
            console.error(`${cmd}: ${e.message}`)
            resolve(1)
        })
        child.on('close', (code) => {
            if (current === child) current = null
            resolve(code ?? 1)
        })
    })
}

/** Whether Ctrl+C (or SIGTERM) came during the run. */
export const wasInterrupted = () => interrupted

/** Something to do before exit, also after Ctrl+C: stop an app, remove containers or files. */
export function onCleanup(fn: () => void | Promise<void>) {
    cleanups.push(fn)
}

/**
 * Ctrl+C reaches the children as well (the terminal sends it to the whole group). This process must not
 * die with them: it still has to clean up. A second Ctrl+C is left to do what it does.
 */
export function trapInterrupts() {
    for (const signal of ['SIGINT', 'SIGTERM'] as const) {
        process.once(signal, () => {
            interrupted = true
            // From a terminal the child has the signal already; from anywhere else it has not.
            current?.kill('SIGINT')
            console.log(`\n${signal}: stopping and cleaning up...`)
        })
    }
}

/** Runs the cleanups (last added first) and exits: 130 after Ctrl+C, else `code`. */
export async function finish(code: number): Promise<never> {
    for (const fn of cleanups.reverse()) {
        try {
            await fn()
        } catch (e) {
            console.error(`cleanup: ${(e as Error).message}`)
        }
    }
    if (interrupted) {
        console.log('Interrupted.')
        process.exit(130)
    }
    process.exit(code)
}
