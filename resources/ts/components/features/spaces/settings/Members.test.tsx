// @vitest-environment jsdom
import { cleanup, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Space } from '@/types/spaces'

const members = [
    { userId: 1, name: 'Me', email: 'me@x.io', role: 'admin' },
    { userId: 2, name: 'Other', email: 'other@x.io', role: 'editor' },
    { userId: 3, name: 'Guest', email: 'guest@x.io', role: 'viewer' },
]

vi.mock('@/hooks', () => ({
    useSpaceMembers: () => ({ data: members }),
    useSetMemberRole: () => ({ mutate: vi.fn() }),
    useRemoveMember: () => ({ mutate: vi.fn() }),
}))
vi.mock('@/components/shared', () => ({
    SettingsSection: ({ children }: { children: React.ReactNode }) => <section>{children}</section>,
}))
vi.mock('@/stores/auth', () => ({ useUser: () => ({ id: 1 }) }))

import { Members } from './Members'

afterEach(() => cleanup())

const space = { id: 1, name: 'Home' } as Space

describe('Members', () => {
    it('gives every row an icon, one role select and one action cell for an admin', () => {
        const { container } = render(<Members space={space} isAdmin />)
        const rows = screen.getAllByRole('listitem')
        expect(rows).toHaveLength(3)
        for (const row of rows) {
            expect(row.querySelectorAll('svg.lucide-user-round')).toHaveLength(1)
            expect(within(row).getAllByRole('combobox')).toHaveLength(1)
        }
        // me: an invisible placeholder; the others: the remove button
        expect(screen.getAllByTestId('member-action-placeholder')).toHaveLength(1)
        expect(within(rows[0]!).getByTestId('member-action-placeholder').getAttribute('aria-hidden')).toBe('true')
        expect(container.querySelectorAll('button[title]')).toHaveLength(2)
    })

    it('stacks identity above the controls on narrow screens and goes single-line from sm', () => {
        render(<Members space={space} isAdmin />)
        for (const row of screen.getAllByRole('listitem')) {
            expect(row.className).toContain('flex-col')
            expect(row.className).toContain('sm:flex-row')
            expect(row.className).not.toContain('flex-wrap')
            // the select takes the free width of line 2 on mobile, fixed width from sm
            const select = within(row).getByRole('combobox')
            expect(select.className).toContain('w-full')
            expect(select.className).toContain('sm:w-32')
        }
    })

    it('shows badges and no buttons to a non-admin', () => {
        render(<Members space={space} isAdmin={false} />)
        expect(screen.queryAllByRole('button')).toHaveLength(0)
        expect(screen.queryAllByRole('combobox')).toHaveLength(0)
        expect(screen.queryAllByTestId('member-action-placeholder')).toHaveLength(0)
    })
})
