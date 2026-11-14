import { describe, expect, it } from 'vitest'
import { localAppConfig, localSeedConfig, parseArgs } from './args.mts'

describe('parseArgs', () => {
    const names = ['full', 'mobile']

    it('reads the config name and the settings', () => {
        expect(parseArgs(['full.json', '--languages=en,ru', '--capture=false', '--detail'], names)).toEqual({
            config: 'full',
            concurrency: '',
            overrides: { languages: 'en,ru', capture: 'false', detail: 'true' },
        })
    })

    it('takes --concurrency out of the settings', () => {
        expect(parseArgs(['--concurrency=8'], names)).toEqual({ config: '', concurrency: '8', overrides: {} })
    })

    it('rejects a bad concurrency and an unknown config', () => {
        expect(() => parseArgs(['--concurrency=0'], names)).toThrow(/--concurrency/)
        expect(() => parseArgs(['nope'], names)).toThrow(/No config "nope"/)
    })
})

describe('localAppConfig', () => {
    const toml = '[server]\napp_url = "http://savvy"\ntimezone = "UTC"\n\n[mailer]\nenabled = true\n'

    it('points the app at the local address and data directory', () => {
        expect(localAppConfig(toml, { url: 'http://127.0.0.1:4000', listen: '127.0.0.1:4000', dataDir: '/tmp/x/data' })).toBe(
            '[server]\napp_url = "http://127.0.0.1:4000"\nlisten = "127.0.0.1:4000"\ntimezone = "UTC"\n\n[mailer]\nenabled = true\n\n[paths]\ndata = "/tmp/x/data"\n',
        )
    })

    it('fails when the config has no app_url to replace', () => {
        expect(() => localAppConfig('[server]\n', { url: 'u', listen: 'l', dataDir: 'd' })).toThrow(/app_url/)
    })
})

describe('localSeedConfig', () => {
    it('writes the manifest into the local data directory', () => {
        expect(localSeedConfig('# seed\nmanifest = "/data/manifest.json"\nmonths = 3\n', '/tmp/x/data/manifest.json')).toBe(
            '# seed\nmanifest = "/tmp/x/data/manifest.json"\nmonths = 3\n',
        )
    })

    it('fails when the seed has no manifest', () => {
        expect(() => localSeedConfig('months = 3\n', '/m')).toThrow(/manifest/)
    })
})
