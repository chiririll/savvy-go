// The command line of `npm run screenshots` and the app's config for a local run. Pure, so it is unit
// tested; main.mts reads the files and the clock.

export interface Args {
    /** A file name of configs/ without .json; empty: the defaults of config.ts. */
    config: string
    /** Browser contexts at a time; empty: the CPUs where the runner runs. */
    concurrency: string
    /** `--key=value` settings, checked by config.ts in the runner. */
    overrides: Record<string, string>
}

export function parseArgs(args: string[], configNames: string[]): Args {
    const overrides: Record<string, string> = {}
    for (const arg of args.filter((a) => a.startsWith('--'))) {
        const [key, ...rest] = arg.slice(2).split('=')
        overrides[key] = rest.length ? rest.join('=') : 'true'
    }
    // Not a setting of the run's config: the runner reads it from the environment.
    const concurrency = overrides.concurrency ?? ''
    delete overrides.concurrency
    if (concurrency && !/^[1-9]\d*$/.test(concurrency)) throw new Error(`--concurrency must be a whole number above 0, not "${concurrency}".`)
    const config = (args.find((a) => !a.startsWith('--')) ?? '').replace(/\.json$/, '')
    if (config && !configNames.includes(config)) {
        throw new Error(`No config "${config}"; there are ${configNames.join(', ') || 'none'} in scripts/screenshots/configs.`)
    }
    return { config, concurrency, overrides }
}

/** The name of a run's directory in screenshots/: the local start time, and the config. */
export function runName(now: Date, config: string): string {
    const pad = (n: number) => String(n).padStart(2, '0')
    const stamp = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}_${pad(now.getHours())}-${pad(now.getMinutes())}-${pad(now.getSeconds())}`
    return config ? `${stamp}_${config}` : stamp
}

const appUrlLine = /^app_url\s*=.*$/m
const manifestLine = /^manifest\s*=.*$/m

/**
 * config.toml (written for the compose app at http://savvy) for an app on this machine: its own address,
 * port and data directory.
 */
export function localAppConfig(toml: string, at: { url: string; listen: string; dataDir: string }): string {
    if (!appUrlLine.test(toml)) throw new Error('scripts/screenshots/config.toml has no app_url line to replace.')
    if (/^\[paths\]/m.test(toml)) throw new Error('scripts/screenshots/config.toml has a [paths] table; the local run adds its own.')
    const server = `app_url = ${JSON.stringify(at.url)}\nlisten = ${JSON.stringify(at.listen)}`
    return `${toml.replace(appUrlLine, server).replace(/\n*$/, '\n')}\n[paths]\ndata = ${JSON.stringify(at.dataDir)}\n`
}

/** seed.toml with the manifest written to `manifest` instead of the container's /data. */
export function localSeedConfig(toml: string, manifest: string): string {
    if (!manifestLine.test(toml)) throw new Error('scripts/screenshots/seed.toml has no manifest line to replace.')
    return toml.replace(manifestLine, `manifest = ${JSON.stringify(manifest)}`)
}
