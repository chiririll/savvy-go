// @vitest-environment jsdom
import { render } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { InfiniteScrollSentinel } from './InfiniteScrollSentinel'

describe('InfiniteScrollSentinel', () => {
    afterEach(() => vi.unstubAllGlobals())

    it('observes against the viewport even inside an overflow-y-auto wrapper that does not scroll', () => {
        const options: IntersectionObserverInit[] = []
        vi.stubGlobal(
            'IntersectionObserver',
            class {
                constructor(_cb: IntersectionObserverCallback, o: IntersectionObserverInit) {
                    options.push(o)
                }
                observe() {}
                disconnect() {}
            },
        )

        render(
            <div style={{ overflowY: 'auto' }}>
                <InfiniteScrollSentinel enabled onVisible={() => {}} />
            </div>,
        )

        expect(options).toHaveLength(1)
        expect(options[0].root).toBeNull()
    })
})
