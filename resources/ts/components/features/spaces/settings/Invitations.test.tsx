// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Space } from '@/types/spaces'

const state = vi.hoisted(() => ({
    list: [] as { id: number; email: string | null; note: string | null; role: string; expiresAt: string; acceptedAt: null }[],
    revoke: vi.fn(),
    token: 'tok123' as string | null,
}))

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key }),
}))
vi.mock('@/components/shared', () => ({
    SettingsSection: ({ children }: { children: React.ReactNode }) => <section>{children}</section>,
}))
vi.mock('@/hooks', () => ({
    useSpaceInvitations: () => ({ data: state.list }),
    useRevokeInvitation: () => ({ mutate: state.revoke }),
}))
vi.mock('@/hooks/use-spaces', () => ({
    useInviteMethods: () => ({ data: { email: false, link: true } }),
}))
vi.mock('../InviteForm', () => ({
    InviteForm: ({ onCreated }: { onCreated: (c: { id: number; link: string | null }) => void }) => (
        <button
            type="button"
            onClick={() => {
                state.list = [
                    ...state.list,
                    { id: 7, email: null, note: 'friend', role: 'editor', expiresAt: '2999-01-01T00:00:00Z', acceptedAt: null },
                ]
                onCreated({ id: 7, link: state.token ? `http://localhost/invite/${state.token}` : null })
            }}
        >
            make
        </button>
    ),
}))

import { Invitations } from './Invitations'

const space = { id: 1, name: 'Home' } as Space

beforeEach(() => {
    state.list = []
    state.token = 'tok123'
    state.revoke.mockReset()
})
afterEach(() => cleanup())

describe('Invitations', () => {
    it('shows an invitation once, with a copy button only for a fresh link', () => {
        const { rerender } = render(<Invitations space={space} />)
        expect(screen.getByText('spaces.invitations.none')).toBeTruthy()

        fireEvent.click(screen.getByText('make'))
        rerender(<Invitations space={space} />)
        expect(screen.getAllByText('friend')).toHaveLength(1)
        expect(screen.getAllByRole('listitem')).toHaveLength(1)
        expect(screen.getByRole('button', { name: /spaces.invitations.copy/ })).toBeTruthy()
        expect(screen.getByText(/spaces.invitations.linkReady/)).toBeTruthy()
    })

    it('has no copy button for an invitation from before a reload', () => {
        state.list = [{ id: 3, email: 'a@x.io', note: null, role: 'viewer', expiresAt: '2999-01-01T00:00:00Z', acceptedAt: null }]
        render(<Invitations space={space} />)
        expect(screen.getAllByRole('listitem')).toHaveLength(1)
        expect(screen.queryByRole('button', { name: /spaces.invitations.copy/ })).toBeNull()
    })

    it('has no copy button for an emailed invitation', () => {
        state.token = null
        const { rerender } = render(<Invitations space={space} />)
        fireEvent.click(screen.getByText('make'))
        rerender(<Invitations space={space} />)
        expect(screen.queryByRole('button', { name: /spaces.invitations.copy/ })).toBeNull()
    })

    it('revokes after the confirm dialog', async () => {
        state.list = [{ id: 3, email: 'a@x.io', note: null, role: 'viewer', expiresAt: '2999-01-01T00:00:00Z', acceptedAt: null }]
        render(<Invitations space={space} />)
        fireEvent.click(screen.getByRole('button', { name: 'spaces.invitations.revoke' }))
        expect(screen.getByText('spaces.invitations.cancelTitle')).toBeTruthy()
        expect(state.revoke).not.toHaveBeenCalled()
        fireEvent.click(screen.getByRole('button', { name: 'spaces.invitations.cancelConfirm' }))
        await waitFor(() => expect(state.revoke).toHaveBeenCalledWith(3))
    })
})
