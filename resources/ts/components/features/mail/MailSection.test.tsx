// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { MailSection } from './MailSection'
import type { MailStatus } from '@/types/mail'

const save = vi.fn()
const check = vi.fn()
let mail: MailStatus | undefined

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key }),
}))
vi.mock('@/hooks', () => ({
    useMailStatus: () => ({ data: mail, isLoading: mail === undefined }),
    useUpdateMail: () => ({ mutate: save, isPending: false }),
    useCheckMail: () => ({ mutateAsync: check, isPending: false }),
    useSendTestMail: () => ({ mutate: vi.fn(), isPending: false }),
}))

vi.mock('@/components/shared', () => ({
    SettingsSection: ({ children }: { children: ReactNode }) => <section>{children}</section>,
    SettingsRowSkeleton: () => <div>loading</div>,
    ToggleRow: ({
        id,
        checked,
        disabled,
        onChange,
    }: {
        id: string
        checked: boolean
        disabled?: boolean
        onChange: (checked: boolean) => void
    }) => <input id={id} type="checkbox" checked={checked} disabled={disabled} onChange={(e) => onChange(e.target.checked)} />,
}))

const base: MailStatus = {
    enabled: true,
    problem: null,
    host: 'smtp.example.com',
    port: 587,
    protocol: 'starttls',
    user: 'savvy',
    from: 'Savvy <savvy@example.com>',
    skipVerify: false,
    timeout: 10,
    passwordSet: true,
    appUrlSet: true,
    writable: true,
}

const field = (id: string) => document.getElementById(id) as HTMLInputElement
const submit = () => act(async () => fireEvent.click(screen.getByRole('button', { name: 'actions.save' })))

beforeEach(() => {
    mail = { ...base }
    check.mockResolvedValue({ ok: true })
})

afterEach(() => {
    cleanup()
    save.mockReset()
    check.mockReset()
})

describe('mail settings form', () => {
    it('shows dots for a stored password until edited, then again if empty', async () => {
        render(<MailSection />)
        expect(field('mail-password').value).toBe('••••••')
        fireEvent.focus(field('mail-password'))
        expect(field('mail-password').value).toBe('')
        fireEvent.blur(field('mail-password'))
        expect(field('mail-password').value).toBe('••••••')
        await submit()
        expect('password' in save.mock.calls[0][0]).toBe(false)
    })

    it('leaves the field empty when no password is stored', () => {
        mail = { ...base, passwordSet: false }
        render(<MailSection />)
        expect(field('mail-password').value).toBe('')
    })

    it('starts from the saved settings', () => {
        render(<MailSection />)
        expect(field('mail-host').value).toBe('smtp.example.com')
        expect(field('mail-port').value).toBe('587')
        expect(field('mail-user').value).toBe('savvy')
        expect(field('mail-timeout').value).toBe('10')
    })

    it('sends every setting, and no password when it was left empty', async () => {
        render(<MailSection />)
        fireEvent.change(field('mail-host'), { target: { value: ' smtp.new.example ' } })
        fireEvent.change(field('mail-timeout'), { target: { value: '30' } })
        await submit()
        expect(save).toHaveBeenCalledTimes(1)
        const input = save.mock.calls[0][0]
        expect(input).toEqual({
            enabled: true,
            host: 'smtp.new.example',
            port: 587,
            protocol: 'starttls',
            user: 'savvy',
            from: 'Savvy <savvy@example.com>',
            skipVerify: false,
            timeout: 30,
        })
        expect('password' in input).toBe(false)
    })

    it('sends a new password when one is typed', async () => {
        render(<MailSection />)
        fireEvent.focus(field('mail-password'))
        fireEvent.change(field('mail-password'), { target: { value: 'new-secret' } })
        await submit()
        expect(save.mock.calls[0][0].password).toBe('new-secret')
    })

    it('clears the stored password when the login is removed', async () => {
        render(<MailSection />)
        fireEvent.change(field('mail-user'), { target: { value: '' } })
        await submit()
        expect(save.mock.calls[0][0]).toMatchObject({ user: '', password: '' })
    })

    it('locks the form and hides Save when config.toml is read-only', () => {
        mail = { ...base, writable: false }
        render(<MailSection />)
        expect(field('mail-host').disabled).toBe(true)
        expect(screen.queryByRole('button', { name: 'actions.save' })).toBeNull()
        expect(screen.getByText('system.mail.configReadOnly')).toBeTruthy()
    })

    it('shows why mail is switched off', () => {
        mail = { ...base, enabled: false, problem: 'mailer.host is empty' }
        render(<MailSection />)
        expect(screen.getByText('system.mail.problem')).toBeTruthy()
    })

    it('shows only the status and the toggle while mail is off', () => {
        mail = { ...base, enabled: false }
        render(<MailSection />)
        expect(field('mail-enabled')).toBeTruthy()
        expect(screen.getByText('system.mail.off')).toBeTruthy()
        expect(field('mail-host')).toBeNull()
        expect(field('mail-timeout')).toBeNull()
        expect(screen.getByRole('button', { name: 'actions.save' })).toBeTruthy()
    })

    it('opens the fields with the toggle and keeps the draft across toggles', () => {
        mail = { ...base, enabled: false }
        render(<MailSection />)
        fireEvent.click(field('mail-enabled'))
        expect(field('mail-host').value).toBe('smtp.example.com')
        fireEvent.change(field('mail-host'), { target: { value: 'smtp.draft.example' } })
        fireEvent.click(field('mail-enabled'))
        expect(field('mail-host')).toBeNull()
        fireEvent.click(field('mail-enabled'))
        expect(field('mail-host').value).toBe('smtp.draft.example')
    })

    it('keeps the fields open when the saved config has a problem', () => {
        mail = { ...base, enabled: false, problem: 'mailer.host is empty' }
        render(<MailSection />)
        expect(field('mail-host')).toBeTruthy()
    })

    it('can still save after switching mail off, sending every setting and no password', async () => {
        render(<MailSection />)
        fireEvent.click(field('mail-enabled'))
        expect(field('mail-host')).toBeNull()
        await submit()
        const input = save.mock.calls[0][0]
        expect(input).toMatchObject({ enabled: false, host: 'smtp.example.com', port: 587, timeout: 10 })
        expect('password' in input).toBe(false)
    })

    it('collapses a read-only form too when mail is not configured', () => {
        mail = { ...base, enabled: false, writable: false }
        render(<MailSection />)
        expect(field('mail-host')).toBeNull()
        expect(screen.getByText('system.mail.configReadOnly')).toBeTruthy()
    })

    it('tests the connection with the draft, without saving', async () => {
        render(<MailSection />)
        fireEvent.change(field('mail-host'), { target: { value: 'smtp.draft.example' } })
        await act(async () => fireEvent.click(screen.getByRole('button', { name: 'system.mail.checkConnection' })))
        expect(check).toHaveBeenCalledTimes(1)
        expect(check.mock.calls[0][0]).toMatchObject({ enabled: true, host: 'smtp.draft.example', port: 587 })
        expect('password' in check.mock.calls[0][0]).toBe(false)
        expect(save).not.toHaveBeenCalled()
    })

    it('has no connection test while the fields are hidden', () => {
        mail = { ...base, enabled: false }
        render(<MailSection />)
        expect(screen.queryByRole('button', { name: 'system.mail.checkConnection' })).toBeNull()
    })

    it('checks the connection before saving enabled settings', async () => {
        render(<MailSection />)
        await submit()
        expect(check).toHaveBeenCalledTimes(1)
        expect(save).toHaveBeenCalledTimes(1)
    })

    it('does not save when the connection check fails', async () => {
        check.mockRejectedValue(new Error('535 authentication failed'))
        render(<MailSection />)
        await submit()
        expect(check).toHaveBeenCalledTimes(1)
        expect(save).not.toHaveBeenCalled()
    })

    it('saves without a check when mail is switched off, so it can always be disabled', async () => {
        check.mockRejectedValue(new Error('unreachable'))
        render(<MailSection />)
        fireEvent.click(field('mail-enabled'))
        await submit()
        expect(check).not.toHaveBeenCalled()
        expect(save.mock.calls[0][0]).toMatchObject({ enabled: false })
    })
})
