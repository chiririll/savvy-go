import { describe, expect, it } from 'vitest'
import { projects, vitestRuns } from './plan.mts'

const base = ['vitest', 'run', '--config', 'vitest.stories.config.ts']

describe('projects', () => {
    it('lists the two English projects, a phone project per other language and dark', () => {
        expect(projects(['en', 'ru', 'de'])).toEqual(['stories-desktop', 'stories-mobile', 'stories-mobile-ru', 'stories-mobile-de', 'stories-desktop-dark'])
    })
})

describe('vitestRuns', () => {
    it('runs the English projects together by default', () => {
        expect(vitestRuns([], false, ['en', 'ru'])).toEqual([[...base, '--project', 'stories-desktop', '--project', 'stories-mobile']])
    })

    it('passes the arguments on, after the default projects', () => {
        expect(vitestRuns(['Foo.stories.tsx', '--reporter=dot'], false, ['en'])).toEqual([
            [...base, '--project', 'stories-desktop', '--project', 'stories-mobile', 'Foo.stories.tsx', '--reporter=dot'],
        ])
    })

    it('leaves the projects to the caller when one is named', () => {
        expect(vitestRuns(['--project', 'stories-mobile-ru'], false, ['en', 'ru'])).toEqual([[...base, '--project', 'stories-mobile-ru']])
        expect(vitestRuns(['--project=stories-desktop-dark'], false, ['en'])).toEqual([[...base, '--project=stories-desktop-dark']])
    })

    it('runs every project on its own for --full, each with the arguments', () => {
        expect(vitestRuns(['--reporter=dot'], true, ['en', 'ru'])).toEqual(
            ['stories-desktop', 'stories-mobile', 'stories-mobile-ru', 'stories-desktop-dark'].map((p) => [...base, '--project', p, '--reporter=dot']),
        )
    })

    it('rejects --full with a project', () => {
        expect(() => vitestRuns(['--project', 'stories-mobile'], true, ['en'])).toThrow(/--full/)
    })
})
