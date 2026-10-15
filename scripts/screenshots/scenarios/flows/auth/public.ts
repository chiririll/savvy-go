// The pages a person without a session opens: sign-in, and the two links an admin hands out (a
// password to set for a new user, an invitation to a space).
import { expect } from 'playwright/test'
import { pages } from '../../../../../resources/ts/app/pages.ts'
import { api, spaceId } from '../../api.ts'
import { scenario } from '../../types.ts'
import { createSpace } from '../onboarding/create-space.ts'

export const publicPages = scenario(
    {
        name: 'auth/public',
        covers: [pages.login.path, pages.setPassword.path, pages.invite.path],
        shots: ['login', 'set-password', 'invite'],
        leaves: 'signed out, on the invitation',
    },
    async ({ page, run, shot }) => {
        await run(createSpace)

        // The links are made while signed in as the admin.
        const user = await api<{ token: string }>(page, 'POST', '/users', { name: 'Sam Rivera', email: 'sam@example.com', role: 'user' })
        const space = await spaceId(page)
        const invitation = await api<{ token: string }>(page, 'POST', `/spaces/${space}/invitations`, { role: 'viewer', email: 'kim@example.com' })
        expect(user.token).toBeTruthy()

        await api(page, 'POST', '/auth/logout')
        await page.goto('/login')
        await expect(page).toHaveURL(/\/login$/)
        await expect(page.locator('input[name="email"]')).toBeVisible()
        await shot('login')

        await page.goto(`/set-password/${user.token}`)
        await expect(page.locator('input[name="password"]')).toBeVisible()
        await shot('set-password')

        await page.goto(`/invite/${invitation.token}`)
        await expect(page.locator('#name')).toBeVisible()
        await shot('invite')
    },
)
