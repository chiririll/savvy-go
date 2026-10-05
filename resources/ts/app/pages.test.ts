import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { pages } from './pages'

// router.tsx is read as text: importing it would load the whole app, which needs a browser.
const router = readFileSync(new URL('./router.tsx', import.meta.url), 'utf8')

describe('pages', () => {
    it('are all routed', () => {
        const missing = Object.keys(pages).filter((name) => !new RegExp(String.raw`pages\.${name}\b`).test(router))
        expect(missing, 'pages that router.tsx does not use').toEqual([])
    })

    it('are the only routes with a path of their own, apart from redirects', () => {
        const literal = router
            .split('\n')
            .filter((line) => /\bpath: '/.test(line) && !/Navigate|Redirect|path: '\*'/.test(line))
        expect(literal, 'routes written out in router.tsx; add them to app/pages.ts').toEqual([])
    })
})
