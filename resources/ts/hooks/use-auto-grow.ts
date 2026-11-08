import { useLayoutEffect, type RefObject } from 'react'

/**
 * Makes a textarea as tall as its text. The cap is the `max-height` of the element (its class), and the least is
 * its `rows`/`min-height`. Measured here because `field-sizing` is not in every browser; give the element
 * `field-sizing-fixed` so the two do not fight. `deps` are what changes the text or the element.
 */
export function useAutoGrow(ref: RefObject<HTMLTextAreaElement | null>, deps: readonly unknown[]) {
    useLayoutEffect(() => {
        const el = ref.current
        if (!el) {
            return
        }
        el.style.height = 'auto'
        const borders = el.offsetHeight - el.clientHeight
        el.style.height = `${el.scrollHeight + borders}px`
    }, deps)
}
