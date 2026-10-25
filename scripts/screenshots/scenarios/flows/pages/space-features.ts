// What a space's settings page does, beyond showing it: one list of invitations, the steps of a sync
// group.
import { expect } from 'playwright/test'
import { pages } from '../../../../../resources/ts/app/pages.ts'
import { api } from '../../api.ts'
import { scenario } from '../../types.ts'
import { visit } from '../../visit.ts'
import { createSpace } from '../onboarding/create-space.ts'

export const spaceInvitations = scenario(
    {
        name: 'pages/space-invitations',
        covers: [],
        shots: ['empty', 'created', 'reloaded'],
        leaves: 'the admin in "Family Budget", on its settings, with one open invitation',
    },
    async ({ page, run, shot }) => {
        await run(createSpace)
        await visit(page, pages.spaceSettings.path)
        await expect(page.getByTestId('invitation-empty')).toBeVisible()
        await shot('empty')

        // A new link shows once, in the list above the form, with a Copy button that exists only now.
        await page.getByTestId('invite-note').fill('Kim from work')
        await page.getByTestId('invite-add').click()
        await expect(page.getByTestId('invitation-list').getByRole('listitem')).toHaveCount(1)
        await expect(page.getByTestId('invite-copy')).toHaveCount(1)
        await shot('created')

        // The server keeps no link: after a reload the invitation is still there, once, with no Copy.
        await page.reload()
        await expect(page.getByTestId('invitation-list').getByRole('listitem')).toHaveCount(1)
        await expect(page.getByTestId('invite-copy')).toHaveCount(0)
        await shot('reloaded')
    },
)

export const syncGroupSteps = scenario(
    {
        name: 'pages/sync-group',
        covers: [],
        shots: ['setup', 'categories', 'tags', 'summary', 'joined'],
        leaves: 'the admin in "Family Budget", which shares categories and tags with "Pets"',
    },
    async ({ page, run, shot }) => {
        await run(createSpace)
        // A second space the admin administers, with a category the first one lacks.
        const pets = await api<{ id: number }>(page, 'POST', '/spaces', {
            name: 'Pets',
            baseCurrency: 'USD',
            accounts: [{ name: 'Cash', type: 'cash', currency: 'USD' }],
            categories: [{ name: 'Vet', type: 'expense', icon: '🐾', color: '#aabbcc' }],
        })
        await visit(page, pages.spaceSettings.path)
        await page.locator(`#sync-space-${pets.id}`).check()
        await expect(page.getByTestId('sync-setup')).toBeEnabled()
        await shot('setup')

        // One decision at a time: categories, tags, then the summary. Back keeps the choices.
        await page.getByTestId('sync-setup').click()
        await expect(page.getByTestId('sync-step-categories')).toBeVisible()
        await shot('categories')
        await page.getByTestId('sync-next').click()
        await expect(page.getByTestId('sync-step-tags')).toBeVisible()
        await shot('tags')
        await page.getByTestId('sync-back').click()
        await expect(page.getByTestId('sync-step-categories')).toBeVisible()
        await page.getByTestId('sync-next').click()
        await page.getByTestId('sync-next').click()
        await expect(page.getByTestId('sync-step-summary')).toBeVisible()
        await shot('summary')

        await page.getByTestId('sync-apply').click()
        await expect(page.getByTestId('sync-apply')).toBeHidden()
        await expect(page.getByTestId('sync-members')).toBeVisible()
        await shot('joined')
    },
)
