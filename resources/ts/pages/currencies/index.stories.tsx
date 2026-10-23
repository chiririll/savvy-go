import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { catalog, currencies } from '@/test/fixtures'
import { hang, reply, settingsApi } from '@/test/api-mocks'
import { useCurrenciesStore } from '@/stores/currencies'

import CurrenciesPage from './index'

const meta = {
    title: 'pages/Currencies',
    component: CurrenciesPage,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/currencies', currencies), reply('get', '/currencies/catalog', catalog), ...settingsApi())
    },
} satisfies Meta<typeof CurrenciesPage>

export default meta
type Story = StoryObj<typeof meta>

// The base currency and one with no fraction and one with three.
export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Japanese Yen')).toBeVisible()
        await expect(canvas.getByText('Kuwaiti Dinar')).toBeVisible()
        await expect(canvas.getByRole('switch', { name: /auto-update currency rates/i })).toBeVisible()
        await expect(canvas.getByRole('button', { name: /new currency/i })).toBeVisible()
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/currencies')) }

export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        // The role comes from the space list, which can land after the rows, and the rows are drawn again then.
        await waitFor(() => expect(canvas.queryByRole('button', { name: /new currency/i })).toBeNull())
        await waitFor(() => expect(canvas.getByRole('switch', { name: /auto-update currency rates/i })).toBeDisabled())
        await expect(await canvas.findByText('Japanese Yen')).toBeVisible()
    },
}

export const OpensTheCreateDialog: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /new currency/i }))
        await expect(await screen.findByRole('dialog')).toBeVisible()
    },
}
