import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { catalog, currencies } from '@/test/fixtures'
import { reply } from '@/test/api-mocks'
import { withForm } from '@/test/form-decorator'

import { CurrencyIdField } from './CurrencyIdField'

const onChange = fn()

const meta = {
    title: 'shared/pickers/CurrencyIdField',
    component: CurrencyIdField,
    parameters: { a11y: overlayA11y },
    decorators: [withForm({ label: 'Currency' })],
    args: { onChange },
    render: (args) => {
        const [value, setValue] = useState(args.value)
        return (
            <CurrencyIdField
                {...args}
                value={value}
                onChange={(id) => {
                    args.onChange(id)
                    setValue(id)
                }}
            />
        )
    },
    beforeEach: ({ msw }) => {
        onChange.mockClear()
        msw.use(reply('get', '/currencies/catalog', catalog), reply('get', '/currencies', currencies))
    },
} satisfies Meta<typeof CurrencyIdField>

export default meta
type Story = StoryObj<typeof meta>

const open = async (canvasElement: HTMLElement) => {
    const trigger = within(canvasElement).getByRole('combobox')
    await waitFor(() => expect(trigger).toBeEnabled())
    await userEvent.click(trigger)
    return trigger
}

export const Empty: Story = {}
export const Selected: Story = {
    args: { value: 3 },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByRole('combobox')).toHaveTextContent('JPY'))
    },
}
export const InRussian: Story = { globals: { locale: 'ru' }, args: { value: 2 } }
export const Disabled: Story = { args: { disabled: true } }

// Only currencies the space already has: the catalog's ones need adding first.
export const OffersOnlyTheSpacesCurrencies: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await expect(await screen.findByText('Kuwaiti Dinar', { exact: false })).toBeVisible()
        await expect(screen.queryByText('Pound Sterling', { exact: false })).toBeNull()
    },
}

// The field hands over the id of the currency, as the budget and debt forms store it.
export const PicksByTheId: Story = {
    play: async ({ canvasElement }) => {
        const trigger = await open(canvasElement)
        await userEvent.click(await screen.findByText('Euro', { exact: false }))
        await expect(onChange).toHaveBeenCalledWith(2)
        await expect(trigger).toHaveTextContent('EUR')
    },
}
