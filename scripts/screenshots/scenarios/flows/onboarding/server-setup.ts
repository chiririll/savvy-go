// Flow 2 (docs/todo-onboarding/04-server-setup.md): the public URL, then the optional steps.
import { expect } from 'playwright/test'
import { pages } from '../../../../../resources/ts/app/pages.ts'
import { scenario } from '../../types.ts'
import { adminSetup } from './admin-setup.ts'

export const serverSetup = scenario(
    {
        name: 'onboarding/server-setup',
        covers: [pages.serverSetup.path],
        shots: ['url', 'options', { name: 'mail', detail: true }, { name: 'sso-presets', detail: true }],
        leaves: 'the admin signed in, the server setup finished, on the offer of two-factor sign-in',
    },
    async ({ page, server, run, shot }) => {
        await run(adminSetup)
        await expect(page).toHaveURL(/\/setup\/server$/)

        // The address the admin opened the app at is offered; keeping it saves it.
        const url = page.locator('#app-url')
        await expect(url).toHaveValue(server.url)
        await shot('url')
        await page.getByTestId('server-setup-continue').click()

        // The optional steps: neither is done yet, so the way out is "skip".
        await expect(page).toHaveURL(/\/setup\/server\/more$/)
        await expect(page.locator('a[href$="/setup/server/mail"]')).toBeVisible()
        await shot('options')

        await page.locator('a[href$="/setup/server/mail"]').click()
        await expect(page).toHaveURL(/\/setup\/server\/mail$/)
        await expect(page.locator('form').first()).toBeVisible()
        await shot('mail')
        await page.locator('a[href$="/setup/server/more"]').click()

        await page.locator('a[href$="/setup/server/sso"]').click()
        await expect(page).toHaveURL(/\/setup\/server\/sso$/)
        await shot('sso-presets')
        await page.locator('a[href$="/setup/server/more"]').click()

        // Skipping ends the flow for good: the server says it is done.
        await page.getByTestId('server-setup-finish').click()
        await expect(page).toHaveURL(/\/setup-2fa$/)
    },
)
