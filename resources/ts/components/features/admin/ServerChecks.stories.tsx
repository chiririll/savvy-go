import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { hang, reply } from '@/test/api-mocks'

import { ServerChecks } from './ServerChecks'

const checks = (list: unknown[]) => reply('get', '/admin/checks', list)

const meta = {
    title: 'features/admin/ServerChecks',
    component: ServerChecks,
    beforeEach: ({ msw }) =>
        msw.use(
            checks([
                { id: 'backup_stale', severity: 'error', value: 12 },
                { id: 'disk_filling', severity: 'warning', value: 91 },
                { id: 'mail_off', severity: 'warning', value: null },
                { id: 'app_url_missing', severity: 'info', value: null },
            ]),
        ),
} satisfies Meta<typeof ServerChecks>

export default meta
type Story = StoryObj<typeof meta>

export const Problems: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Last server backup is 12 days old')).toBeVisible()
        await expect(canvas.getByText('Data disk is 91% full')).toBeVisible()
        // Each problem links to the page that fixes it.
        await expect(canvas.getByRole('link', { name: /open monitoring/i })).toHaveAttribute('href', '/admin/monitoring')
        await expect(canvas.getByRole('link', { name: /set up email/i })).toHaveAttribute('href', '/admin/system')
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
export const InDarkTheme: Story = { globals: { theme: 'dark' } }

export const OneDayOld: Story = {
    beforeEach: ({ msw }) => msw.use(checks([{ id: 'backup_stale', severity: 'warning', value: 1 }])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Last server backup is 1 day old')).toBeVisible()
    },
}

export const AllGood: Story = {
    beforeEach: ({ msw }) => msw.use(checks([])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/everything looks fine/i)).toBeVisible()
    },
}

// Nothing is shown until the server answers.
export const Loading: Story = {
    beforeEach: ({ msw }) => msw.use(hang('get', '/admin/checks')),
    play: async ({ canvasElement }) => {
        await expect(canvasElement.firstElementChild).toBeEmptyDOMElement()
    },
}
