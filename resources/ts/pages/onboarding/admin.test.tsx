// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import AdminSetupPage from './admin'

const register = vi.fn()
const status = vi.fn()
const verifyBootstrap = vi.fn()
const toastError = vi.fn()

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key }),
}))
vi.mock('@/api', () => ({ authApi: { status: () => status(), verifyBootstrap: (p: string) => verifyBootstrap(p) } }))
vi.mock('@/stores/auth', () => ({
    useAuthStore: (select: (state: object) => unknown) => select({ register }),
}))
vi.mock('@/components/shared', () => ({ LanguageSwitcher: () => null }))
vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: (m: string) => toastError(m) } }))

function renderPage(queryClient = new QueryClient()) {
    return {
        queryClient,
        ...render(
            <QueryClientProvider client={queryClient}>
                <MemoryRouter initialEntries={['/setup']}>
                    <Routes>
                        <Route path="/setup" element={<AdminSetupPage />} />
                        <Route path="/setup-2fa" element={<div>next</div>} />
                        <Route path="/login" element={<div>login</div>} />
                    </Routes>
                </MemoryRouter>
            </QueryClientProvider>,
        ),
    }
}

const type = (label: string, value: string) =>
    fireEvent.change(screen.getByLabelText(label), { target: { value } })

async function toAdminScreen(password = 'ABCDE') {
    await screen.findByLabelText('setup.bootstrapPassword')
    type('setup.bootstrapPassword', password)
    fireEvent.click(screen.getByRole('button', { name: 'setup.createAdmin' }))
    await screen.findByLabelText('setup.name')
}

function fillAdmin() {
    type('setup.name', 'Jane')
    type('email', 'jane@example.com')
    type('password', 'secret1')
    type('setup.confirmPassword', 'secret1')
    fireEvent.click(screen.getByRole('button', { name: 'setup.submit' }))
}

beforeEach(() => {
    verifyBootstrap.mockResolvedValue(undefined)
    status.mockResolvedValue({
        needs_registration: true,
        bootstrap_hint: { environment: 'docker', file: '/data/bootstrap-password' },
    })
})

afterEach(() => {
    cleanup()
    register.mockReset()
    verifyBootstrap.mockReset()
    toastError.mockClear()
})

describe('admin setup', () => {
    it('starts with the bootstrap password and the instructions for the environment', async () => {
        renderPage()
        await screen.findByLabelText('setup.bootstrapPassword')
        expect(screen.getByText('docker logs <container>')).toBeTruthy()
        expect(screen.queryByLabelText('setup.name')).toBeNull()
    })

    it('does not offer a restore until there is one', async () => {
        renderPage()
        await screen.findByLabelText('setup.bootstrapPassword')
        expect(screen.queryByText('setup.restore')).toBeNull()
    })

    it('does not go on without a bootstrap password', async () => {
        renderPage()
        await screen.findByLabelText('setup.bootstrapPassword')
        fireEvent.click(screen.getByRole('button', { name: 'setup.createAdmin' }))
        await screen.findByText('setup.validation.bootstrapPassword')
        expect(screen.queryByLabelText('setup.name')).toBeNull()
    })

    it('checks the password before asking for the admin', async () => {
        verifyBootstrap.mockRejectedValue({
            message: 'x',
            details: { bootstrap_password: ['The bootstrap password is missing or wrong.'] },
        })
        renderPage()
        await screen.findByLabelText('setup.bootstrapPassword')
        type('setup.bootstrapPassword', 'WRONG')
        fireEvent.click(screen.getByRole('button', { name: 'setup.createAdmin' }))
        await screen.findByText('The bootstrap password is missing or wrong.')
        expect(verifyBootstrap).toHaveBeenCalledWith('WRONG')
        expect(screen.queryByLabelText('setup.name')).toBeNull()
    })

    it('shows the error of a wrong password, goes back to it and creates no user', async () => {
        register.mockRejectedValue({
            message: 'x',
            details: { bootstrap_password: ['The bootstrap password is missing or wrong.'] },
        })
        renderPage()
        await toAdminScreen('WRONG')
        fillAdmin()
        await screen.findByLabelText('setup.bootstrapPassword')
        expect(screen.getByText('The bootstrap password is missing or wrong.')).toBeTruthy()
        expect(register).toHaveBeenCalledTimes(1)
        expect(screen.queryByText('next')).toBeNull()
    })

    it('signs the admin in and moves on after registering', async () => {
        register.mockResolvedValue(undefined)
        const { queryClient } = renderPage()
        const invalidate = vi.spyOn(queryClient, 'invalidateQueries')
        await toAdminScreen('RIGHT')
        fillAdmin()
        await screen.findByText('next')
        expect(register).toHaveBeenCalledWith({
            bootstrap_password: 'RIGHT',
            name: 'Jane',
            email: 'jane@example.com',
            password: 'secret1',
        })
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ['auth', 'status'] })
    })

    it('goes to the login page when the server already has users', async () => {
        status.mockResolvedValue({ needs_registration: false, bootstrap_hint: null })
        renderPage()
        await waitFor(() => expect(screen.getByText('login')).toBeTruthy())
    })
})
