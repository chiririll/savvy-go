// `npm run screenshots` in Docker: docker-compose.screenshots.yml, removed afterwards whether or not the
// run worked. Needs only Node and Docker. Nothing is bind-mounted, the files are copied out of the
// container, so any Docker daemon works (the active context).
import { mkdirSync, readFileSync } from 'node:fs'
import path from 'node:path'
import { onCleanup, run } from '../runner.mts'
import type { Run } from './main.mts'

export async function runInDocker({ root, here, runName, env }: Run): Promise<number> {
    // The app's config files go to the container as variables: a mount would name a path on the daemon's host.
    const composeEnv = {
        ...process.env,
        ...env,
        APP_CONFIG: readFileSync(path.join(here, 'config.toml'), 'utf8'),
        APP_SEED: readFileSync(path.join(here, 'seed.toml'), 'utf8'),
    }
    const compose = (...args: string[]) => run('docker', ['compose', '-f', 'docker-compose.screenshots.yml', ...args], { cwd: root, env: composeEnv })

    onCleanup(async () => {
        await compose('down', '-v', '--remove-orphans')
    })
    const code = await compose('up', '--build', '--abort-on-container-exit', '--exit-code-from', 'shots')
    // Before `down`, which removes the container: it is stopped by now, but cp still reads it. Also after a
    // failed run, which leaves FAILED.png.
    mkdirSync(path.join(root, 'screenshots'), { recursive: true })
    if ((await compose('cp', `shots:/out/${runName}`, 'screenshots')) !== 0) console.error('Could not copy the screenshots out of the container.')
    return code
}
