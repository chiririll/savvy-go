// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { SyncChange, SyncDiff } from '@/types/spaces'

const state = vi.hoisted(() => ({
    data: undefined as unknown,
    apply: vi.fn(),
    load: vi.fn(),
    reset: vi.fn(),
}))

const categories = vi.hoisted(() => [{ type: 'expense', name: '#GROCERIES' }])

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string, o?: Record<string, unknown>) => (o?.step ? `${key}:${o.step}` : key) }),
}))
vi.mock('@/hooks', () => ({
    useSyncDiff: () => ({ mutate: state.load, reset: state.reset, data: state.data, isPending: false, isError: false }),
    useApplySync: () => ({ mutate: state.apply, isPending: false }),
    useCategories: () => ({ data: categories }),
}))
vi.mock('@/lib/localized-name', () => ({
    localizeDefaultName: (name: string) => (name.startsWith('#') ? `Translated ${name.slice(1)}` : name),
}))
vi.mock('@/components/shared/ResponsiveDialog', () => ({
    ResponsiveDialog: ({ title, description, footer, children }: Record<string, ReactNode>) => (
        <div role="dialog">
            <h2>{title}</h2>
            <p>{description}</p>
            {children}
            <div>{footer}</div>
        </div>
    ),
}))
// Radix selects need a browser; a native select does the same job here.
vi.mock('@/components/ui/select', () => ({
    Select: ({ value, onValueChange, children }: { value: string; onValueChange: (v: string) => void; children: ReactNode }) => (
        <select value={value} onChange={(e) => onValueChange(e.target.value)}>
            {children}
        </select>
    ),
    SelectTrigger: () => null,
    SelectValue: () => null,
    SelectContent: ({ children }: { children: ReactNode }) => <>{children}</>,
    SelectItem: ({ value, children }: { value: string; children: ReactNode }) => <option value={value}>{children}</option>,
}))

import { SyncReconcileDialog } from './SyncReconcileDialog'

const change: SyncChange = { add_space_ids: [1, 2], categories: true, tags: true }

const diff = {
    spaces: [
        { id: 1, name: 'Home' },
        { id: 2, name: 'Trip' },
    ],
    categories: [
        {
            type: 'expense',
            name: '#HOUSING',
            present: [{ spaceId: 1, icon: '🏠', color: '#111', isDefault: false, transactions: 4 }],
            missing: [2],
        },
    ],
    tags: [{ name: '#SHARED', present: [{ spaceId: 1, transactions: 0 }], missing: [2] }],
    defaults: [],
} as unknown as SyncDiff

const next = () => fireEvent.click(screen.getByRole('button', { name: 'spaces.sync.dialog.next' }))
const back = () => fireEvent.click(screen.getByRole('button', { name: 'spaces.sync.dialog.back' }))
const apply = () => screen.getByRole('button', { name: 'spaces.sync.dialog.apply' })
const keepSelect = () => screen.getAllByRole('combobox')[0] as HTMLSelectElement

beforeEach(() => {
    state.data = diff
    state.apply.mockReset()
})
afterEach(() => cleanup())

describe('SyncReconcileDialog', () => {
    it('walks the categories, tags and summary, and applies only on the last', () => {
        render(<SyncReconcileDialog change={change} onClose={vi.fn()} />)
        expect(screen.getByText(/Translated HOUSING/)).toBeTruthy()
        expect(screen.queryByRole('button', { name: 'spaces.sync.dialog.apply' })).toBeNull()
        next()
        expect(screen.getByText(/#Translated SHARED/)).toBeTruthy()
        next()
        expect(apply().hasAttribute('disabled')).toBe(false)
        fireEvent.click(apply())
        expect(state.apply).toHaveBeenCalledTimes(1)
        expect(state.apply.mock.calls[0]![0]).toMatchObject({ decisions: [{ keep: true }, { keep: true }] })
    })

    it('keeps the choices when going back', () => {
        render(<SyncReconcileDialog change={change} onClose={vi.fn()} />)
        fireEvent.change(keepSelect(), { target: { value: 'drop' } })
        next()
        back()
        expect(keepSelect().value).toBe('drop')
    })

    it('asks where the transactions go, and holds Apply back until that is chosen', () => {
        render(<SyncReconcileDialog change={change} onClose={vi.fn()} />)
        fireEvent.change(keepSelect(), { target: { value: 'drop' } })
        next()
        // the merge step: a deleted category with transactions needs a successor
        expect(screen.getByText('spaces.sync.dialog.successor')).toBeTruthy()
        expect(screen.getByRole('button', { name: 'spaces.sync.dialog.next' }).hasAttribute('disabled')).toBe(true)
        fireEvent.change(screen.getByRole('combobox'), { target: { value: '#GROCERIES' } })
        next()
        next()
        fireEvent.click(apply())
        expect(state.apply.mock.calls[0]![0].decisions[0]).toMatchObject({ keep: false, successor: '#GROCERIES' })
    })

    it('skips straight to the summary when only defaults differ', () => {
        state.data = { ...diff, categories: [], tags: [], defaults: [{ type: 'expense', defaults: [{ spaceId: 1, name: '#OTHER' }] }] }
        render(<SyncReconcileDialog change={change} onClose={vi.fn()} />)
        expect(screen.getByText('spaces.sync.dialog.summaryDefaults')).toBeTruthy()
        expect(apply()).toBeTruthy()
    })
})
