import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { catalog, currencies } from '@/test/fixtures'
import { fail, hang, reply } from '@/test/api-mocks'
import { withForm } from '@/test/form-decorator'

import { CurrencySelect, type CurrencySelectValue } from './CurrencySelect'

const meta = {
    title: 'shared/CurrencySelect',
    component: CurrencySelect,
    args: { onChange: () => {} },
    decorators: [withForm({ label: 'Currency' })],
    render: (args) => {
        const [value, setValue] = useState(args.value)
        return <CurrencySelect {...args} value={value} onChange={(next: CurrencySelectValue) => setValue(next)} />
    },
    beforeEach: ({ msw }) => {
        msw.use(reply('get', '/currencies/catalog', catalog), reply('get', '/currencies', currencies))
    },
} satisfies Meta<typeof CurrencySelect>

export default meta
type Story = StoryObj<typeof meta>

const open = async (canvasElement: HTMLElement) => {
    const trigger = within(canvasElement).getByRole('combobox')
    await waitFor(() => expect(trigger).toBeEnabled())
    await userEvent.click(trigger)
    return trigger
}

export const Empty: Story = {}

export const Existing: Story = { args: { value: { source: 'existing', id: 3 } } }

export const FromCatalog: Story = { args: { value: { source: 'catalog', code: 'GBP' } } }

export const Disabled: Story = { args: { disabled: true } }

export const CustomPlaceholder: Story = { args: { placeholder: 'Pick one' } }

export const Loading: Story = {
    beforeEach: ({ msw }) => msw.use(hang('get', '/currencies')),
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('combobox')).toBeDisabled()
    },
}

export const LoadFailed: Story = {
    beforeEach: ({ msw }) => msw.use(fail('get', '/currencies')),
}

export const Opened: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await expect(await screen.findByText('Japanese Yen', { exact: false })).toBeVisible()
        await expect(screen.getByText('Pound Sterling', { exact: false })).toBeVisible()
    },
}

export const OnlyExisting: Story = {
    args: { allowCatalog: false },
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await expect(await screen.findByText('Euro', { exact: false })).toBeVisible()
        await expect(screen.queryByText('Pound Sterling', { exact: false })).toBeNull()
    },
}

export const Searched: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await userEvent.type(await screen.findByRole('textbox'), 'ruble')
        await expect(screen.getByText('Russian Ruble', { exact: false })).toBeVisible()
        await expect(screen.queryByText('Euro', { exact: false })).toBeNull()
    },
}

export const NothingFound: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await userEvent.type(await screen.findByRole('textbox'), 'zzzz')
        await expect(await screen.findByText(/no currenc|nothing/i)).toBeVisible()
    },
}

export const Picked: Story = {
    play: async ({ canvasElement }) => {
        const trigger = await open(canvasElement)
        await userEvent.click(await screen.findByText('Euro', { exact: false }))
        await expect(trigger).toHaveTextContent('EUR')
    },
}

export const Invalid: Story = {
    decorators: [withForm({ label: 'Currency', error: 'Pick a currency' })],
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Pick a currency')).toBeVisible()
    },
}

export const Russian: Story = {
    globals: { locale: 'ru' },
    args: { value: { source: 'existing', id: 2 } },
}
