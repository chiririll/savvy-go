import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { accounts, categories, currencies } from '@/test/fixtures'
import { reply } from '@/test/api-mocks'
import { useCurrenciesStore } from '@/stores/currencies'

import ImportSettingsPage from './import'

const meta = {
    title: 'pages/settings/Import',
    component: ImportSettingsPage,
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/accounts', accounts), reply('get', '/categories', categories))
    },
} satisfies Meta<typeof ImportSettingsPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('heading', { level: 1 })).toBeVisible()
        // The wizard starts with the file.
        await expect(canvasElement.querySelector('input[type="file"]')).not.toBeNull()
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }

// A viewer cannot import, and is told so instead of being shown the wizard.
export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('heading', { level: 2 })).toBeVisible()
        await expect(canvasElement.querySelector('input[type="file"]')).toBeNull()
    },
}
