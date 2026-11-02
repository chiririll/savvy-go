// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

const webauthn = vi.hoisted(() => ({ supported: true }))
vi.mock('@simplewebauthn/browser', () => ({ browserSupportsWebAuthn: () => webauthn.supported }))
vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key }),
}))
vi.mock('@/components/shared', () => ({
    SettingsSection: ({ children }: { children: ReactNode }) => <section>{children}</section>,
    ResponsiveDialog: () => null,
    Confirm: () => null,
}))
vi.mock('@/components/providers/ReadOnlyProvider', () => ({ useReadOnly: () => false }))
vi.mock('@/hooks', () => ({
    isPasskeyDomainSupported: () => true,
    useWebauthnCredentials: () => ({ data: [], isLoading: false }),
    useRegisterPasskey: () => ({ isPending: false, mutateAsync: vi.fn() }),
    useDeletePasskey: () => ({ isPending: false, mutate: vi.fn() }),
}))

import { PasskeysSection } from './PasskeysSection'

afterEach(() => cleanup())

describe('PasskeysSection', () => {
    it('warns and disables adding when the browser has no WebAuthn', () => {
        webauthn.supported = false
        render(<PasskeysSection />)
        expect(screen.getByRole('alert').textContent).toBe('security.passkeys.unsupported')
        expect((screen.getByRole('button', { name: /passkeys.add/ }) as HTMLButtonElement).disabled).toBe(true)
    })

    it('lets the user add a passkey in a supporting browser', () => {
        webauthn.supported = true
        render(<PasskeysSection />)
        expect(screen.queryByRole('alert')).toBeNull()
        expect((screen.getByRole('button', { name: /passkeys.add/ }) as HTMLButtonElement).disabled).toBe(false)
    })
})
