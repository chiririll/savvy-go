// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'

const webauthn = vi.hoisted(() => ({ supported: true }))
vi.mock('@simplewebauthn/browser', () => ({ browserSupportsWebAuthn: () => webauthn.supported }))

import { UserSettingsTabs } from './UserSettingsTabs'

afterEach(() => cleanup())

const links = () => screen.getAllByRole('link').map((a) => a.getAttribute('href'))

describe('UserSettingsTabs', () => {
    it('lists Security, Passkeys and API, no Preferences', () => {
        webauthn.supported = true
        render(<MemoryRouter><UserSettingsTabs /></MemoryRouter>)
        expect(links()).toEqual(['/settings/security', '/settings/passkeys', '/settings/api'])
    })

    it('hides Passkeys when the browser cannot use them', () => {
        webauthn.supported = false
        render(<MemoryRouter><UserSettingsTabs /></MemoryRouter>)
        expect(links()).toEqual(['/settings/security', '/settings/api'])
    })
})
