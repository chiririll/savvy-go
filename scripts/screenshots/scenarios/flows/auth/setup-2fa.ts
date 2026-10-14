// The offer of two-factor sign-in that follows the first admin's setup (pages/auth/setup-2fa.tsx):
// enabled with a code from an authenticator app, or skipped.
import { expect } from 'playwright/test'
import { pages } from '../../../../../resources/ts/app/pages.ts'
import { totp } from '../../totp.ts'
import { scenario } from '../../types.ts'
import { serverSetup } from '../onboarding/server-setup.ts'

export const setup2fa = scenario(
    {
        name: 'auth/setup-2fa',
        covers: [pages.setup2fa.path],
        shots: ['intro', 'qr', { name: 'verify', detail: true }, { name: 'wrong-code', detail: true }, 'recovery-codes'],
        leaves: 'two-factor sign-in on for the admin, no space yet: the create-space modal is open',
    },
    async ({ page, run, shot }) => {
        await run(serverSetup)
        await expect(page).toHaveURL(/\/setup-2fa$/)
        await shot('intro')

        await page.getByTestId('2fa-enable').click()
        const secret = page.getByTestId('2fa-secret')
        await expect(secret).toBeVisible()
        // The secret and its QR code differ every run: hidden in the shot.
        await shot('qr', { mask: [page.getByTestId('2fa-qr'), secret] })
        const key = (await secret.innerText()).trim()
        await page.getByTestId('2fa-continue').click()

        const code = page.locator('input[autocomplete="one-time-code"]')
        await expect(code).toBeVisible()
        await shot('verify')

        // A wrong code is refused and nothing is enabled.
        await code.pressSequentially('000000')
        await page.getByTestId('2fa-verify').click()
        await expect(page.locator('[data-sonner-toast][data-type="error"]')).toBeVisible()
        await expect(page.getByTestId('2fa-verify')).toBeVisible()
        await shot('wrong-code', { toasts: true })

        // The field is made of slots and does not take a new value over an old one: empty it, then type.
        await code.press('Control+A')
        await code.press('Backspace')
        await code.pressSequentially(totp(key))
        await page.getByTestId('2fa-verify').click()
        const codes = page.getByTestId('2fa-recovery-codes')
        await expect(codes).toBeVisible()
        expect(await codes.locator('div').count()).toBeGreaterThan(0)
        await shot('recovery-codes', { mask: [codes] })

        await page.getByTestId('2fa-done').click()
        await expect(page.locator('#space-name')).toBeVisible()
    },
)

/** The same offer, skipped: the way most of the scenarios after it go on. */
export const setup2faSkipped = scenario(
    {
        name: 'auth/setup-2fa-skipped',
        covers: [],
        shots: [],
        leaves: 'the admin signed in with no space: the create-space modal is open',
    },
    async ({ page, run }) => {
        await run(serverSetup)
        await expect(page).toHaveURL(/\/setup-2fa$/)
        await page.getByTestId('2fa-skip').click()
        // A user without a space gets the create-space modal at once, and cannot close it.
        await expect(page.locator('#space-name')).toBeVisible()
    },
)
