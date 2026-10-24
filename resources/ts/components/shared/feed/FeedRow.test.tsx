// @vitest-environment jsdom
import { cleanup, render, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

const state = vi.hoisted(() => ({ mobile: false }))

vi.mock('@/hooks', () => ({
    useIsMobile: () => state.mobile,
    useLongPress: () => ({ handlers: {}, consume: () => false, mark: () => {} }),
}))
vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key }),
}))

import { RowActions, ROW_ACTIONS_TRIGGER_SIZE } from '@/components/shared/table/RowActions'
import { FeedRow } from './FeedRow'

afterEach(() => {
    cleanup()
    state.mobile = false
})

const actions = () => <RowActions onEdit={() => {}} />

function row(props: { hasActions?: boolean; withActions?: boolean }) {
    return (
        <FeedRow
            icon="x"
            title="Row"
            amount="10.00"
            hasActions={props.hasActions}
            actions={props.withActions === false ? undefined : actions}
        />
    )
}

/** Everything after the amount column, i.e. what decides the amount's right offset. */
function trailing(container: HTMLElement) {
    const amount = within(container).getByText('10.00').parentElement!
    const items: Element[] = []
    for (let el = amount.nextElementSibling; el; el = el.nextElementSibling) items.push(el)
    return items
}

describe('FeedRow actions column', () => {
    it('reserves the dots width on a row without actions when actions are provided', () => {
        const { container } = render(row({ hasActions: false }))
        const [spacer, ...rest] = trailing(container)
        expect(rest).toHaveLength(0)
        expect(spacer!.getAttribute('data-testid')).toBe('feed-row-actions-spacer')
        expect(spacer!.classList.contains(ROW_ACTIONS_TRIGGER_SIZE)).toBe(true)
    })

    it('puts the amount at the same offset with and without actions', () => {
        const { container: withDots } = render(row({ hasActions: true }))
        const dots = trailing(withDots)
        const trigger = dots[0]!.querySelector('button')!
        const { container: without } = render(row({ hasActions: false }))
        const spacer = trailing(without)[0]!

        expect(dots).toHaveLength(1)
        expect(trigger.classList.contains(ROW_ACTIONS_TRIGGER_SIZE)).toBe(true)
        expect(spacer.classList.contains(ROW_ACTIONS_TRIGGER_SIZE)).toBe(true)
        // same negative margin on both wrappers
        expect(spacer.classList.contains('-mr-1')).toBe(dots[0]!.classList.contains('-mr-1'))
    })

    it('adds no spacer when the list passes no actions', () => {
        const { container } = render(row({ withActions: false }))
        expect(trailing(container)).toHaveLength(0)
    })

    it('adds no spacer on mobile, where the menu opens by long press', () => {
        state.mobile = true
        const { container } = render(row({ hasActions: false }))
        expect(trailing(container)).toHaveLength(0)
    })
})
