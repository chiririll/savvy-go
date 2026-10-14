// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { MailSection } from './MailSection'
import type { MailStatus } from '@/types/mail'

const save = vi.fn()
let mail: MailStatus | undefined

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key }),
}))
vi.mock('@/hooks', () => ({
    useMailStatus: () => ({ data: mail, isLoading: mail === undefined }),
    useUpdateMail: () => ({ mutate: save, isPending: false }),
    useSendTestMail: () => ({ mutate: vi.fn(), isPending: false }),
}))

vi.mock('@/components/shared', () => ({
    SettingsSection: ({ children }: { children: ReactNode }) => <section>{children}</section>,
    SettingsRowSkeleton: () => <div>loading</div>,
    ToggleRow: ({ id, checked, disabled }: { id: string; checked: boolean; disabled?: boolean }) => (
        <input id={id} type="checkbox" checked={checked} disabled={disabled} readOnly />
    ),
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
const submit = () => fireEvent.click(screen.getByRole('button', { name: 'actions.save' }))

beforeEach(() => {
    mail = { ...base }
})

afterEach(() => {
    cleanup()
    save.mockReset()
})

describe('mail settings form', () => {
    it('shows dots for a stored password, which go away when editing and return if left empty', () => {
        render(<MailSection />)
        expect(field('mail-password').value).toBe('••••••')
        fireEvent.focus(field('mail-password'))
        expect(field('mail-password').value).toBe('')
        fireEvent.blur(field('mail-password'))
        expect(field('mail-password').value).toBe('••••••')
        submit()
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

    it('sends every setting, and no password when it was left empty', () => {
        render(<MailSection />)
        fireEvent.change(field('mail-host'), { target: { value: ' smtp.new.example ' } })
        fireEvent.change(field('mail-timeout'), { target: { value: '30' } })
        submit()
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

    it('sends a new password when one is typed', () => {
        render(<MailSection />)
        fireEvent.focus(field('mail-password'))
        fireEvent.change(field('mail-password'), { target: { value: 'new-secret' } })
        submit()
        expect(save.mock.calls[0][0].password).toBe('new-secret')
    })

    it('clears the stored password when the login is removed', () => {
        render(<MailSection />)
        fireEvent.change(field('mail-user'), { target: { value: '' } })
        submit()
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
})
