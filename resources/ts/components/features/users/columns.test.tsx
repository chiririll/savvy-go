// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { createUserColumns } from './columns'
import type { User } from '@/types/users'

vi.mock('@/lib/i18n', () => ({ default: { t: (key: string) => key } }))
vi.mock('@/components/shared', () => ({
    UserAvatar: () => null,
    RowActions: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}))
vi.mock('@/components/ui/dropdown-menu', () => ({
    DropdownMenuItem: ({ children }: { children: ReactNode }) => <button>{children}</button>,
}))

const user: User = {
    id: 2,
    name: 'Ann',
    email: 'ann@example.com',
    role: 'user',
    isInactive: false,
    isSsoOnly: false,
    emailVerified: false,
    createdAt: '2026-01-01',
}

const noop = () => {}
const columns = (showUnverified?: boolean) =>
    createUserColumns({
        onDelete: noop,
        onEdit: noop,
        onResetPassword: noop,
        onSetEmailVerified: noop,
        showUnverified,
    })

const renderCell = (index: number, showUnverified?: boolean, u: User = user) => {
    const col = columns(showUnverified)[index] as { cell: (ctx: unknown) => ReactNode }
    render(<>{col.cell({ row: { original: u } })}</>)
}

afterEach(cleanup)

describe('unconfirmed badge', () => {
    it('is hidden unless asked for, even for an unverified user', () => {
        renderCell(0)
        expect(screen.queryByText('pages:users.emailUnverified')).toBeNull()
    })

    it('is shown for an unverified user when asked for', () => {
        renderCell(0, true)
        expect(screen.getByText('pages:users.emailUnverified')).toBeTruthy()
    })

    it('is not shown for a verified user', () => {
        renderCell(0, true, { ...user, emailVerified: true })
        expect(screen.queryByText('pages:users.emailUnverified')).toBeNull()
    })
})

describe('mark verified menu item', () => {
    it('follows the same flag', () => {
        renderCell(3)
        expect(screen.queryByText('pages:users.markVerified')).toBeNull()
        cleanup()
        renderCell(3, true)
        expect(screen.getByText('pages:users.markVerified')).toBeTruthy()
    })

    it('still lets an admin withdraw a verified address when shown', () => {
        renderCell(3, true, { ...user, emailVerified: true })
        expect(screen.getByText('pages:users.markUnverified')).toBeTruthy()
    })
})
