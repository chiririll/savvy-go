// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { SpaceGate } from './SpaceGate'

let role = 'user'

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key }),
}))
vi.mock('@/components/features/spaces/CreateSpaceModal', () => ({
    CreateSpaceModal: ({ open, onOpenChange, dismissible = true }: { open: boolean; onOpenChange: (open: boolean) => void; dismissible?: boolean }) =>
        open ? (
            <div role="dialog" data-dismissible={String(dismissible)}>
                <button onClick={() => onOpenChange(false)}>close</button>
            </div>
        ) : null,
}))
vi.mock('@/hooks/use-spaces', () => ({ useSpaces: () => ({ data: [], isPending: false }) }))
vi.mock('@/stores/auth', () => ({ useUser: () => ({ name: 'Jane', role }) }))
vi.mock('@/stores/space', () => ({
    useSpaceStore: (select: (state: object) => unknown) => select({ currentId: null, setCurrentId: () => {} }),
}))

function renderGate() {
    return render(
        <QueryClientProvider client={new QueryClient()}>
            <SpaceGate>app</SpaceGate>
        </QueryClientProvider>,
    )
}

afterEach(cleanup)

describe('a user with no spaces', () => {
    it('gets the create-space modal, which cannot be dismissed', async () => {
        role = 'user'
        renderGate()
        expect(screen.queryByText('app')).toBeNull()
        expect((await screen.findByRole('dialog')).getAttribute('data-dismissible')).toBe('false')
    })

    it('can open it again from the page', async () => {
        role = 'user'
        renderGate()
        fireEvent.click(await screen.findByText('close'))
        expect(screen.queryByRole('dialog')).toBeNull()
        fireEvent.click(screen.getByRole('button', { name: 'spaces.create' }))
        expect(screen.getByRole('dialog')).toBeTruthy()
    })
})

describe('a guest with no spaces', () => {
    it('sees the message only', () => {
        role = 'guest'
        renderGate()
        expect(screen.getByText('spaces.noneGuest')).toBeTruthy()
        expect(screen.queryByRole('button')).toBeNull()
        expect(screen.queryByRole('dialog')).toBeNull()
    })
})
