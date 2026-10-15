import { describe, expect, it } from 'vitest'

import { chooseInviteMethod } from './invite-method'

describe('chooseInviteMethod', () => {
    it('lets the inviter pick when both work', () => {
        expect(chooseInviteMethod({ email: true, link: true }, 'link')).toBe('link')
        expect(chooseInviteMethod({ email: true, link: true }, 'email')).toBe('email')
    })

    it('is email only when the server requires verified addresses', () => {
        expect(chooseInviteMethod({ email: true, link: false }, 'link')).toBe('email')
    })

    it('is link only without mail', () => {
        expect(chooseInviteMethod({ email: false, link: true }, 'email')).toBe('link')
    })

    it('is nothing when neither works or nothing is known yet', () => {
        expect(chooseInviteMethod({ email: false, link: false }, 'email')).toBeNull()
        expect(chooseInviteMethod(undefined, 'email')).toBeNull()
    })
})
