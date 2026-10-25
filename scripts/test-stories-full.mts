// The story language and theme matrix (docs/development.md), one vitest run per project, one after another.
// `vitest run --project "stories-*"` starts a browser for every project at once; on a CI-sized host the last one never
// ran a story and the run hung with no output. A project run on its own takes about 80 s.
import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.join(path.dirname(fileURLToPath(import.meta.url)), '..')
const locales = fs
    .readdirSync(path.join(root, 'resources/ts/locales'), { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && entry.name !== 'en')
    .map((entry) => entry.name)

const projects = ['stories-desktop', 'stories-mobile', ...locales.map((locale) => `stories-mobile-${locale}`), 'stories-desktop-dark']
// `--list` prints the projects as JSON for the CI matrix (.github/workflows/ci-frontend-stories.yaml), one runner each.
if (process.argv[2] === '--list') {
    console.log(JSON.stringify(projects))
    process.exit(0)
}

const extra = process.argv.slice(2)

const failed: string[] = []
for (const project of projects) {
    console.log(`\n=== ${project} ===`)
    const run = spawnSync('npx', ['vitest', 'run', '--project', project, ...extra], { cwd: root, stdio: 'inherit', shell: true })
    if (run.status !== 0) failed.push(project)
}

if (failed.length > 0) {
    console.error(`\nFailed: ${failed.join(', ')}`)
    process.exit(1)
}
