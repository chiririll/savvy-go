// scripts/package.json "screenshots" (npm run screenshots in the root): runs
// docker-compose.screenshots.yml and removes what it created afterwards, whether or not
// the run worked. Needs only Node and Docker.
import { spawnSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..')
const compose = (...args) =>
    spawnSync('docker', ['compose', '-f', 'docker-compose.screenshots.yml', ...args], { cwd: root, stdio: 'inherit' })

const run = compose('up', '--build', '--abort-on-container-exit', '--exit-code-from', 'shots')
compose('down', '-v', '--remove-orphans')
process.exit(run.status ?? 1)
