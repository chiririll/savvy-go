import { describe, expect, it, vi } from 'vitest'

import type { InviteResult, SpaceRole } from '@/types/spaces'
import { sendInvitations } from './send-invitations'
import type { DraftInvitation } from './draft'

const item = (key: string, email: string | null, role: SpaceRole = 'editor'): DraftInvitation => ({ key, email, role })

describe('sendInvitations', () => {
    it('emails the addressed ones and builds a link for the rest', async () => {
        const invite = vi.fn(async (_space: number, _role: SpaceRole, email?: string): Promise<InviteResult> =>
            email ? { id: 1, emailSent: true } : { id: 2, token: 'abc', emailSent: false })

        const out = await sendInvitations(9, [item('a', 'bob@example.org'), item('b', null, 'viewer')], invite, 'https://savvy.example')

        expect(invite).toHaveBeenNthCalledWith(1, 9, 'editor', 'bob@example.org')
        expect(invite).toHaveBeenNthCalledWith(2, 9, 'viewer', undefined)
        expect(out).toMatchObject([
            { emailed: true, link: null, error: null },
            { emailed: false, link: 'https://savvy.example/invite/abc', error: null },
        ])
    })

    it('reports the one that failed and still sends the others', async () => {
        const invite = vi.fn(async (_space: number, _role: SpaceRole, email?: string): Promise<InviteResult> => {
            if (email === 'bad@example.org') throw { message: 'This address was invited a moment ago.' }
            return { id: 1, emailSent: true }
        })

        const out = await sendInvitations(9, [item('a', 'bad@example.org'), item('b', 'ok@example.org')], invite, 'https://x')

        expect(out[0]).toMatchObject({ emailed: false, link: null, error: 'This address was invited a moment ago.' })
        expect(out[1]).toMatchObject({ emailed: true, error: null })
    })

    it('does nothing for an empty list', async () => {
        const invite = vi.fn()
        expect(await sendInvitations(9, [], invite, 'https://x')).toEqual([])
        expect(invite).not.toHaveBeenCalled()
    })
})
