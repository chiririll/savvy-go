// Flow 1 (docs/todo-onboarding/03-admin-setup.md): the bootstrap password, then the first admin.
import { expect } from 'playwright/test'
import { pages } from '../../../../../resources/ts/app/pages.ts'
import { scenario } from '../../types.ts'

const status = async (url: string) => (await (await fetch(`${url}/api/auth/status`)).json()) as { needs_registration: boolean }

export const adminSetup = scenario(
    {
        name: 'onboarding/admin-setup',
        covers: [pages.setup.path],
        shots: ['bootstrap-password', { name: 'wrong-password', detail: true }, { name: 'restore-backup', detail: true }, 'create-admin', { name: 'validation-errors', detail: true }],
        leaves: 'the first admin signed in, on the server setup',
    },
    async ({ page, server, shot }) => {
        // An empty server sends every address to the setup.
        await page.goto('/')
        await expect(page).toHaveURL(/\/setup$/)
        const password = page.locator('#bootstrap-password')
        await expect(password).toBeVisible()
        await shot('bootstrap-password')

        // A wrong password is refused on the first screen, and no one is created.
        await password.fill('AAAAA-AAAAA-AAAAA-AAAAA-AAAAA-AAAAA')
        await page.getByTestId('setup-create-admin').click()
        await expect(page.locator('p.text-destructive')).toBeVisible()
        await expect(password).toBeVisible()
        expect((await status(server.url)).needs_registration).toBe(true)
        await shot('wrong-password')

        // The right one, to restore a backup instead (that flow has its own scenario).
        await password.fill(server.bootstrapPassword())
        await page.getByTestId('setup-restore').click()
        await expect(page.locator('#backup-file')).toBeVisible()
        await shot('restore-backup')
        await page.getByTestId('setup-back').click()

        await expect(password).toBeVisible()
        await page.getByTestId('setup-create-admin').click()
        const email = page.locator('input[name="email"]')
        await expect(email).toBeVisible()
        await shot('create-admin')

        // Empty fields: every one says what is missing.
        await page.locator('button[type="submit"]').click()
        await expect(page.locator('[id$="-form-item-message"]').first()).toBeVisible()
        await shot('validation-errors')

        await page.locator('input[name="name"]').fill('Alex Morgan')
        await email.fill('alex@example.com')
        await page.locator('input[name="password"]').fill('correct-horse-1')
        await page.locator('input[name="password_confirmation"]').fill('correct-horse-1')
        await page.locator('button[type="submit"]').click()

        // The admin is signed in and goes on to the server setup; the password is spent.
        await expect(page).toHaveURL(/\/setup\/server$/)
        expect((await status(server.url)).needs_registration).toBe(false)
    },
)
