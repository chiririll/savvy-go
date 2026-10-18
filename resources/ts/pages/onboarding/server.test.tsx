// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import ServerSetupPage from './server'

const update = vi.fn()
const createProvider = vi.fn()
const checkAuth = vi.fn()
let settings: Record<string, unknown> | undefined
let user: { role: string } | null
let loading = false
let mail: { enabled: boolean } | undefined
let providers: unknown[] | undefined

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key }),
}))
vi.mock('@/hooks', () => ({
    useInstanceSettings: () => ({ data: settings, isLoading: settings === undefined }),
    useUpdateSettings: () => ({ mutateAsync: (data: object) => update(data), isPending: false }),
    useMailStatus: () => ({ data: mail }),
    useIdentityProviders: () => ({ data: providers }),
    useCreateIdentityProvider: (redirectTo: string) => ({
        mutate: (data: object) => createProvider(data, redirectTo),
        isPending: false,
    }),
}))
vi.mock('@/stores/auth', () => ({
    useUser: () => user,
    useAuthLoading: () => loading,
    useAuthStore: (select: (state: object) => unknown) => select({ checkAuth }),
}))
vi.mock('@/components/shared', () => ({ LanguageSwitcher: () => null }))
vi.mock('@/components/features/mail', () => ({ MailSection: () => <div>mail-section</div> }))
vi.mock('@/components/features/sso', () => ({
    PresetChooser: ({ onSelect }: { onSelect: (preset: string) => void }) => (
        <button onClick={() => onSelect('google')}>preset-google</button>
    ),
    SelectedPresetBanner: () => <div>banner</div>,
    IdentityProviderForm: ({ onSubmit }: { onSubmit: (data: object) => void }) => (
        <button onClick={() => onSubmit({ preset: 'google' })}>submit-provider</button>
    ),
}))
vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

function renderPage(path = '/setup/server') {
    return render(
        <MemoryRouter initialEntries={[path]}>
            <Routes>
                <Route path="/setup/server/*" element={<ServerSetupPage />} />
                <Route path="/setup-2fa" element={<div>next</div>} />
                <Route path="/" element={<div>home</div>} />
                <Route path="/login" element={<div>login</div>} />
            </Routes>
        </MemoryRouter>,
    )
}

const urlInput = () => screen.getByLabelText('serverSetup.url.label') as HTMLInputElement
const click = (name: string) => fireEvent.click(screen.getByRole('button', { name }))

beforeEach(() => {
    user = { role: 'admin' }
    loading = false
    mail = { enabled: false }
    providers = []
    settings = { app_url: '', config_writable: true, onboarding_server_done: false }
    update.mockResolvedValue({})
})

afterEach(() => {
    cleanup()
    update.mockReset()
    createProvider.mockReset()
    checkAuth.mockReset()
})

describe('server setup: who sees it', () => {
    it('is not offered to non-admins', () => {
        user = { role: 'editor' }
        renderPage()
        expect(screen.getByText('home')).toBeTruthy()
    })

    it('goes to the login page without a session', () => {
        user = null
        renderPage()
        expect(screen.getByText('login')).toBeTruthy()
    })

    it('restores the session when opened cold (reload, new tab) before deciding', () => {
        user = null
        loading = true
        renderPage()
        expect(checkAuth).toHaveBeenCalled()
        expect(screen.queryByText('login')).toBeNull()
        expect(screen.queryByText('home')).toBeNull()
    })

    it('does not check the session again when the admin was just created', () => {
        renderPage()
        expect(checkAuth).not.toHaveBeenCalled()
    })

    it('is not shown again once it is done', () => {
        settings = { ...settings, onboarding_server_done: true }
        renderPage()
        expect(screen.getByText('home')).toBeTruthy()
    })

    it('is not offered on a server that never had it', () => {
        settings = { ...settings, onboarding_server_done: null }
        renderPage()
        expect(screen.getByText('home')).toBeTruthy()
    })
})

describe('step 1: the public URL', () => {
    it('fills the field with the browser address if no URL is set', () => {
        renderPage()
        expect(urlInput().value).toBe(window.location.origin)
    })

    it('keeps the URL that is already set', () => {
        settings = { ...settings, app_url: 'https://savvy.example.com' }
        renderPage()
        expect(urlInput().value).toBe('https://savvy.example.com')
    })

    it('saves the URL and goes on to the optional steps', async () => {
        renderPage()
        fireEvent.change(urlInput(), { target: { value: ' https://savvy.example.com ' } })
        click('serverSetup.continue')
        await screen.findByText('serverSetup.optionsTitle')
        expect(update).toHaveBeenCalledWith({ app_url: 'https://savvy.example.com' })
    })

    it('does not save an unchanged URL', async () => {
        settings = { ...settings, app_url: 'https://savvy.example.com' }
        renderPage()
        click('serverSetup.continue')
        await screen.findByText('serverSetup.optionsTitle')
        expect(update).not.toHaveBeenCalled()
    })

    it('stays on the step when saving fails', async () => {
        update.mockRejectedValue(new Error('no'))
        renderPage()
        click('serverSetup.continue')
        await waitFor(() => expect(update).toHaveBeenCalled())
        expect(screen.queryByText('serverSetup.optionsTitle')).toBeNull()
    })

    it('cannot go on with an empty URL', () => {
        renderPage()
        fireEvent.change(urlInput(), { target: { value: '  ' } })
        expect((screen.getByRole('button', { name: 'serverSetup.continue' }) as HTMLButtonElement).disabled).toBe(true)
    })

    it('shows a note instead of an input when the config is read-only', async () => {
        settings = { ...settings, config_writable: false }
        renderPage()
        expect(screen.getByText('serverSetup.url.readOnly')).toBeTruthy()
        expect(screen.queryByLabelText('serverSetup.url.label')).toBeNull()
        click('serverSetup.continue')
        await screen.findByText('serverSetup.optionsTitle')
        expect(update).not.toHaveBeenCalled()
    })
})

describe('step 2: the optional steps', () => {
    it('lists them as links to their own screens', () => {
        renderPage('/setup/server/more')
        expect(screen.getByRole('link', { name: /serverSetup\.mail\.title/ }).getAttribute('href')).toBe(
            '/setup/server/mail',
        )
        expect(screen.getByRole('link', { name: /serverSetup\.sso\.title/ }).getAttribute('href')).toBe(
            '/setup/server/sso',
        )
    })

    it('marks nothing as done at first and offers to skip', () => {
        renderPage('/setup/server/more')
        expect(screen.queryByText('serverSetup.done')).toBeNull()
        expect(screen.getByRole('button', { name: 'serverSetup.skip' })).toBeTruthy()
    })

    it('marks the configured ones as done and offers to finish', () => {
        mail = { enabled: true }
        renderPage('/setup/server/more')
        expect(screen.getByRole('link', { name: /serverSetup\.mail\.title/ }).textContent).toContain('serverSetup.done')
        expect(screen.getByRole('link', { name: /serverSetup\.sso\.title/ }).textContent).not.toContain(
            'serverSetup.done',
        )
        expect(screen.getByRole('button', { name: 'serverSetup.finish' })).toBeTruthy()
    })

    it('marks single sign-on as done once a provider exists', () => {
        providers = [{ id: 1 }]
        renderPage('/setup/server/more')
        expect(screen.getByRole('link', { name: /serverSetup\.sso\.title/ }).textContent).toContain('serverSetup.done')
    })

    it('opens a step on its own screen and returns to the list', async () => {
        renderPage('/setup/server/more')
        fireEvent.click(screen.getByRole('link', { name: /serverSetup\.mail\.title/ }))
        await screen.findByText('mail-section')
        expect(screen.queryByText('serverSetup.optionsTitle')).toBeNull()
        fireEvent.click(screen.getByRole('link', { name: 'serverSetup.backToList' }))
        await screen.findByText('serverSetup.optionsTitle')
    })

    it('adds an identity provider and returns to the list', () => {
        renderPage('/setup/server/sso')
        click('preset-google')
        click('submit-provider')
        expect(createProvider).toHaveBeenCalledWith({ preset: 'google' }, '/setup/server/more')
    })

    it('finishing marks it done and moves on', async () => {
        mail = { enabled: true }
        renderPage('/setup/server/more')
        click('serverSetup.finish')
        await screen.findByText('next')
        expect(update).toHaveBeenCalledWith({ onboarding_server_done: true })
    })

    it('skipping marks it done too', async () => {
        renderPage('/setup/server/more')
        click('serverSetup.skip')
        await screen.findByText('next')
        expect(update).toHaveBeenCalledWith({ onboarding_server_done: true })
    })

    it('goes back to the required step', async () => {
        renderPage('/setup/server/more')
        fireEvent.click(screen.getByRole('link', { name: 'serverSetup.back' }))
        await screen.findByLabelText('serverSetup.url.label')
    })
})
