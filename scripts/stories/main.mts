// `npm run stories` and `npm run stories:full` (docs/development.md): every Storybook story as a test in
// Chromium, in Docker (the default; scripts/stories/Dockerfile) or here (--local, needs `npx playwright
// install chromium`).
//
//   npm run stories                                    English and light, desktop and phone
//   npm run stories -- --local Foo.stories.tsx         one file, here
//   npm run stories -- --project stories-mobile-ru     the projects named, instead of the English ones
//   npm run stories:full -- --reporter=dot             the language and theme matrix, project by project
//   node scripts/stories/main.mts --list               the projects as JSON (the CI matrix)
//
// Anything else goes to vitest. In Docker the same script runs in the container with --local.
import { randomBytes } from 'node:crypto'
import { readdirSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { finish, modeOrExit, onCleanup, run, trapInterrupts, wasInterrupted } from '../runner.mts'
import { projects, vitestRuns } from './plan.mts'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..')
const locales = readdirSync(path.join(root, 'resources/ts/locales'), { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)

const argv = process.argv.slice(2)
if (argv[0] === '--list') {
    console.log(JSON.stringify(projects(locales)))
    process.exit(0)
}

const { mode, rest: withFull } = modeOrExit(argv)
const full = withFull.includes('--full')
const rest = withFull.filter((a) => a !== '--full')
trapInterrupts()

if (mode === 'local') {
    let runs: string[][]
    try {
        runs = vitestRuns(rest, full, locales)
    } catch (e) {
        console.error((e as Error).message)
        process.exit(2)
    }
    const failed: string[] = []
    for (const args of runs) {
        const project = full ? args[args.indexOf('--project') + 1] : ''
        if (project) console.log(`\n=== ${project} ===`)
        if ((await run('npx', args, { cwd: root })) !== 0) failed.push(project || 'stories')
        if (wasInterrupted()) break
    }
    if (full && failed.length) console.error(`\nFailed: ${failed.join(', ')}`)
    await finish(failed.length ? 1 : 0)
}

// Docker: build the image (cached after the first time), then run this script in it.
const image = 'savvy-go-stories'
if ((await run('docker', ['build', '-f', 'scripts/stories/Dockerfile', '-t', image, '.'], { cwd: root })) !== 0) {
    console.error('Could not build the stories image.')
    await finish(1)
}
// Named, so Ctrl+C can remove it: the signal does not always get through `docker run` to vitest.
const name = `${image}-${randomBytes(4).toString('hex')}`
onCleanup(async () => {
    if (wasInterrupted()) await run('docker', ['rm', '-f', name], { stdio: 'ignore' })
})
const env = ['STORYBOOK_CONTRAST', 'CI'].filter((k) => process.env[k]).flatMap((k) => ['-e', `${k}=${process.env[k]}`])
// Colours for a terminal, without a TTY in the container (which would turn Ctrl+C into the container's).
if (process.stdout.isTTY) env.push('-e', 'FORCE_COLOR=1')
const code = await run('docker', [
    'run', '--rm', '--init', '--name', name,
    // Chromium recommends sharing the host's IPC namespace in Docker.
    '--ipc=host',
    ...env,
    image, 'node', 'scripts/stories/main.mts', '--local', ...(full ? ['--full'] : []), ...rest,
], { cwd: root })
await finish(code)
