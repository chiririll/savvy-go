// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ServerSetupGate } from './ServerSetupGate'

let user: { role: string } | null
let settings: { data?: Record<string, unknown>; isPending: boolean }
const useSettings = vi.fn()

vi.mock('@/stores/auth', () => ({ useUser: () => user }))
vi.mock('@/hooks/use-settings', () => ({ useSettings: (enabled: boolean) => useSettings(enabled) }))

function renderGate() {
    return render(
        <MemoryRouter initialEntries={['/transactions']}>
            <Routes>
                <Route path="/transactions" element={<ServerSetupGate><div>app</div></ServerSetupGate>} />
                <Route path="/setup/server" element={<div>server setup</div>} />
            </Routes>
        </MemoryRouter>,
    )
}

beforeEach(() => {
    user = { role: 'admin' }
    settings = { data: { onboarding_server_done: false }, isPending: false }
    useSettings.mockImplementation(() => settings)
})

afterEach(() => {
    cleanup()
    useSettings.mockReset()
})

describe('server setup gate', () => {
    it('sends an admin back to the server setup while it is pending, e.g. after a reload', () => {
        renderGate()
        expect(screen.getByText('server setup')).toBeTruthy()
    })

    it('lets an admin through once it is done', () => {
        settings = { data: { onboarding_server_done: true }, isPending: false }
        renderGate()
        expect(screen.getByText('app')).toBeTruthy()
    })

    it('does nothing on a server that never had it', () => {
        settings = { data: { onboarding_server_done: null }, isPending: false }
        renderGate()
        expect(screen.getByText('app')).toBeTruthy()
    })

    it('never asks anyone but an admin', () => {
        user = { role: 'editor' }
        renderGate()
        expect(screen.getByText('app')).toBeTruthy()
        expect(useSettings).toHaveBeenCalledWith(false)
    })

    it('waits for the settings of an admin instead of flashing the app', () => {
        settings = { data: undefined, isPending: true }
        renderGate()
        expect(screen.queryByText('app')).toBeNull()
    })

    it('lets an admin through when the settings cannot be read', () => {
        settings = { data: undefined, isPending: false }
        renderGate()
        expect(screen.getByText('app')).toBeTruthy()
    })
})
