// Connecting an AI app (Claude web, ...) over OAuth: the admin opens the registration window in
// Administration → Security, the app registers itself and sends the browser to the consent page
// (pages/oauth/consent.tsx), where the user picks the spaces and allows. The app is the
// repository's own probe (cmd/oauth-probe, what `go tool mage probe:oauth` runs), as a binary.
import { expect } from 'playwright/test'
import { pages } from '../../../../../resources/ts/app/pages.ts'
import { startOauthProbe } from '../../probe.ts'
import { scenario } from '../../types.ts'
import { visit } from '../../visit.ts'
import { createSpace } from '../onboarding/create-space.ts'

export const oauthApp = scenario(
    {
        name: 'auth/oauth-app',
        covers: [pages.oauthConsent.path],
        shots: ['registration-closed', 'registration-open', 'consent', 'redirected', 'connected'],
        leaves: 'the admin in the space "Family Budget", on Security, with one connected app and registration closed again',
    },
    async ({ page, server, run, shot }) => {
        await run(createSpace)

        await visit(page, pages.adminSecurity.path)
        const toggle = page.getByTestId('mcp-registration')
        await expect(toggle).toBeVisible()
        await shot('registration-closed')

        // The app can only register while the window is open.
        const opened = page.waitForResponse((r) => r.url().endsWith('/api/admin/oauth/registration') && r.request().method() === 'POST')
        await toggle.click()
        expect((await opened).ok()).toBe(true)
        // "Open until HH:MM" is the time of the run plus five minutes.
        const status = page.locator('main p.text-sm.text-muted-foreground').first()
        await expect(status).toContainText(/\d/)
        await shot('registration-open', { mask: [status] })

        const probe = startOauthProbe(server)
        try {
            await page.goto(await probe.authorizeUrl)
            await expect(page).toHaveURL(/\/oauth\/consent\?/)
            const allow = page.getByTestId('oauth-allow')
            await expect(allow).toBeVisible()
            // Nothing is allowed until a space is picked.
            await expect(allow).toBeDisabled()
            await page.locator('[id^="oauth-space-"][role="checkbox"]').first().click()
            await expect(allow).toBeEnabled()
            // The port the app listens on differs every run.
            await shot('consent', { mask: [page.getByText(/127\.0\.0\.1/)] })

            await allow.click()
            await expect(page).toHaveURL(/127\.0\.0\.1:\d+\/callback/)
            await expect(page.getByText('You can close this tab')).toBeVisible()
            await shot('redirected')

            // The probe goes on with the tokens, then says it is done.
            await probe.finished
        } finally {
            probe.stop()
        }

        await visit(page, pages.adminSecurity.path)
        const app = page.locator('[data-testid^="mcp-app-"]')
        await expect(app).toHaveCount(1)
        await expect(app).toContainText('oauth-probe')
        // Its address, port and time of registration differ every run.
        await shot('connected', { mask: [app.locator('dl')] })
    },
)
