// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { SpaceGate } from './SpaceGate'

const create = vi.fn().mockResolvedValue({})
let role = 'user'

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key }),
}))
vi.mock('@/api/spaces', () => ({ spacesApi: { create: (name: string) => create(name) } }))
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

afterEach(() => {
    cleanup()
    create.mockClear()
})

describe('a user with no spaces', () => {
    it('can create one in a click, named after them', async () => {
        role = 'user'
        renderGate()
        expect(screen.queryByText('app')).toBeNull()
        fireEvent.click(screen.getByRole('button', { name: 'spaces.create' }))
        await waitFor(() => expect(create).toHaveBeenCalledWith('Jane'))
    })
})

describe('a guest with no spaces', () => {
    it('sees the message only', () => {
        role = 'guest'
        renderGate()
        expect(screen.getByText('spaces.noneGuest')).toBeTruthy()
        expect(screen.queryByRole('button')).toBeNull()
    })
})
