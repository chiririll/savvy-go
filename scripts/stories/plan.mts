// Which vitest runs `npm run stories` makes (docs/development.md). Pure, so it is unit tested.

const config = ['vitest', 'run', '--config', 'vitest.stories.config.ts']
const english = ['stories-desktop', 'stories-mobile']

/** Every story project: English at both sizes, every other language at phone size, then dark. */
export function projects(locales: string[]): string[] {
    return [...english, ...locales.filter((l) => l !== 'en').map((l) => `stories-mobile-${l}`), 'stories-desktop-dark']
}

const namesProject = (args: string[]) => args.some((a) => a === '--project' || a.startsWith('--project='))

/**
 * The npx argument lists to run one after another. Without --full: one run of the English projects, or
 * of the projects named in `args`. With --full: every project on its own, because starting a browser for
 * every project at once hung on a CI-sized host (a project alone takes about 80 s).
 */
export function vitestRuns(args: string[], full: boolean, locales: string[]): string[][] {
    if (full) {
        if (namesProject(args)) throw new Error('--full runs every project; leave out --project, or leave out --full.')
        return projects(locales).map((p) => [...config, '--project', p, ...args])
    }
    if (namesProject(args)) return [[...config, ...args]]
    return [[...config, ...english.flatMap((p) => ['--project', p]), ...args]]
}
