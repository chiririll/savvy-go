import { describe, expect, it } from 'vitest'
import { takeMode } from './runner.mts'

describe('takeMode', () => {
    it('defaults to docker', () => {
        expect(takeMode(['full'], {})).toEqual({ mode: 'docker', rest: ['full'] })
    })

    it('takes --local and --docker out of the arguments', () => {
        expect(takeMode(['--local', 'full', '--concurrency=2'], {})).toEqual({ mode: 'local', rest: ['full', '--concurrency=2'] })
        expect(takeMode(['full', '--docker'], { SAVVY_RUNNER: 'local' })).toEqual({ mode: 'docker', rest: ['full'] })
    })

    it('falls back to SAVVY_RUNNER', () => {
        expect(takeMode([], { SAVVY_RUNNER: 'local' }).mode).toBe('local')
        expect(takeMode([], { SAVVY_RUNNER: ' Docker ' }).mode).toBe('docker')
        expect(takeMode([], { SAVVY_RUNNER: '' }).mode).toBe('docker')
    })

    it('rejects both flags and an unknown SAVVY_RUNNER', () => {
        expect(() => takeMode(['--local', '--docker'], {})).toThrow(/either --docker or --local/)
        expect(() => takeMode([], { SAVVY_RUNNER: 'podman' })).toThrow(/SAVVY_RUNNER/)
    })

    it('leaves what follows a bare -- alone', () => {
        expect(takeMode(['--local', '--', '--docker'], {})).toEqual({ mode: 'local', rest: ['--', '--docker'] })
    })
})
