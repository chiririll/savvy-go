// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import AdminSetupPage from './admin'

const register = vi.fn()
const status = vi.fn()
const verifyBootstrap = vi.fn()
const restore = vi.fn()
const toastError = vi.fn()

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key }),
}))
vi.mock('@/api', () => ({
    authApi: { status: () => status(), verifyBootstrap: (p: string) => verifyBootstrap(p) },
    onboardingApi: { restore: (p: string, f: File, k?: string) => restore(p, f, k) },
}))
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
                        <Route path="/setup/server" element={<div>next</div>} />
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
    restore.mockReset()
    toastError.mockClear()
})

describe('admin setup', () => {
    it('starts with the bootstrap password and the instructions for the environment', async () => {
        renderPage()
        await screen.findByLabelText('setup.bootstrapPassword')
        expect(screen.getByText('docker logs <container>')).toBeTruthy()
        expect(screen.queryByLabelText('setup.name')).toBeNull()
    })

    describe('restore a backup', () => {
        const backup = () => new File(['x'], 'backup.zip')

        async function toRestoreScreen(password = 'RIGHT') {
            await screen.findByLabelText('setup.bootstrapPassword')
            type('setup.bootstrapPassword', password)
            fireEvent.click(screen.getByRole('button', { name: 'setup.restore' }))
            await screen.findByLabelText('setup.backupFile')
        }

        const chooseFile = (file: File) =>
            fireEvent.change(screen.getByLabelText('setup.backupFile'), { target: { files: [file] } })

        it('checks the password before asking for the file', async () => {
            verifyBootstrap.mockRejectedValue({
                message: 'x',
                details: { bootstrap_password: ['The bootstrap password is missing or wrong.'] },
            })
            renderPage()
            await screen.findByLabelText('setup.bootstrapPassword')
            type('setup.bootstrapPassword', 'WRONG')
            fireEvent.click(screen.getByRole('button', { name: 'setup.restore' }))
            await screen.findByText('The bootstrap password is missing or wrong.')
            expect(screen.queryByLabelText('setup.backupFile')).toBeNull()
        })

        it('sends the password and the file, then goes to the login page', async () => {
            restore.mockResolvedValue(undefined)
            const { queryClient } = renderPage()
            const invalidate = vi.spyOn(queryClient, 'invalidateQueries')
            await toRestoreScreen()
            const file = backup()
            chooseFile(file)
            fireEvent.click(screen.getByRole('button', { name: 'setup.restoreSubmit' }))
            await screen.findByText('login')
            expect(restore).toHaveBeenCalledWith('RIGHT', file, undefined)
            expect(invalidate).toHaveBeenCalledWith({ queryKey: ['auth', 'status'] })
        })

        it('does not send without a file', async () => {
            renderPage()
            await toRestoreScreen()
            fireEvent.click(screen.getByRole('button', { name: 'setup.restoreSubmit' }))
            await screen.findByText('setup.validation.backupFile')
            expect(restore).not.toHaveBeenCalled()
        })

        it('asks for the app key of a Laravel database only', async () => {
            restore.mockResolvedValue(undefined)
            renderPage()
            await toRestoreScreen()
            chooseFile(backup())
            expect(screen.queryByLabelText('setup.appKey')).toBeNull()
            const laravel = new File(['x'], 'database.sqlite')
            chooseFile(laravel)
            type('setup.appKey', ' base64:abc ')
            fireEvent.click(screen.getByRole('button', { name: 'setup.restoreSubmit' }))
            await screen.findByText('login')
            expect(restore).toHaveBeenCalledWith('RIGHT', laravel, 'base64:abc')
        })

        it('shows why the backup was refused and stays on the screen', async () => {
            restore.mockRejectedValue({ message: 'x', details: { file: ['The backup has no users.'] } })
            renderPage()
            await toRestoreScreen()
            chooseFile(backup())
            fireEvent.click(screen.getByRole('button', { name: 'setup.restoreSubmit' }))
            await screen.findByText('The backup has no users.')
            expect(screen.queryByText('login')).toBeNull()
        })

        it('goes back to the first screen', async () => {
            renderPage()
            await toRestoreScreen()
            fireEvent.click(screen.getByRole('button', { name: 'setup.back' }))
            await screen.findByLabelText('setup.bootstrapPassword')
        })
    })

    it('does not go on without a bootstrap password', async () => {
        renderPage()
        await screen.findByLabelText('setup.bootstrapPassword')
        fireEvent.click(screen.getByRole('button', { name: 'setup.createAdmin' }))
        await screen.findByText('setup.validation.bootstrapPassword')
        expect(screen.queryByLabelText('setup.name')).toBeNull()
    })

    it('Enter on the first screen goes on and does not validate the admin fields', async () => {
        renderPage()
        await screen.findByLabelText('setup.bootstrapPassword')
        type('setup.bootstrapPassword', 'RIGHT')
        fireEvent.submit(screen.getByLabelText('setup.bootstrapPassword').closest('form')!)
        await screen.findByLabelText('setup.name')
        expect(verifyBootstrap).toHaveBeenCalledWith('RIGHT')
        expect(screen.queryByText('setup.validation.name')).toBeNull()
        expect(screen.queryByText('setup.validation.email')).toBeNull()
        expect(screen.queryByText('setup.validation.passwordMin')).toBeNull()
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

    it('does not bounce to the login page once the status says registration is over', async () => {
        // The first admin exists now: the refreshed status no longer asks for registration.
        register.mockImplementation(async () => {
            status.mockResolvedValue({ needs_registration: false, bootstrap_hint: null })
        })
        renderPage()
        await toAdminScreen('RIGHT')
        fillAdmin()
        await screen.findByText('next')
        await new Promise((resolve) => setTimeout(resolve, 50))
        expect(screen.getByText('next')).toBeTruthy()
        expect(screen.queryByText('login')).toBeNull()
    })

    it('goes to the login page when the server already has users', async () => {
        status.mockResolvedValue({ needs_registration: false, bootstrap_hint: null })
        renderPage()
        await waitFor(() => expect(screen.getByText('login')).toBeTruthy())
    })
})
