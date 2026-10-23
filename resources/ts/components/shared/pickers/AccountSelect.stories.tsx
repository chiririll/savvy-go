import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { accounts, currencies } from '@/test/fixtures'
import { hang, reply } from '@/test/api-mocks'
import { overlayA11y } from '@/test/a11y'
import { withForm } from '@/test/form-decorator'
import { useCurrenciesStore } from '@/stores/currencies'

import { AccountSelect } from './AccountSelect'

const meta = {
    title: 'shared/pickers/AccountSelect',
    component: AccountSelect,
    parameters: { a11y: overlayA11y },
    decorators: [withForm({ label: 'Account' })],
    args: { onChange: () => {} },
    render: (args) => {
        const [value, setValue] = useState(args.value)
        return <AccountSelect {...args} value={value} onChange={setValue} />
    },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/accounts', accounts))
    },
} satisfies Meta<typeof AccountSelect>

export default meta
type Story = StoryObj<typeof meta>

const open = async (canvasElement: HTMLElement) => {
    const trigger = within(canvasElement).getByRole('combobox')
    await userEvent.click(trigger)
    return trigger
}

export const Empty: Story = {}
export const Selected: Story = { args: { value: 3 } }
export const Disabled: Story = { args: { disabled: true, value: 1 } }
export const CustomPlaceholder: Story = { args: { placeholder: 'From which account?' } }
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/accounts')) }
// No space chosen yet: nothing to list, and the select stays shut.
export const WaitingForSpace: Story = {
    args: { spaceId: null },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('combobox')).toBeDisabled()
    },
}

export const ListsEveryAccount: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await waitFor(() => expect(screen.getAllByRole('option')).toHaveLength(4))
    },
}

// Balances are shown in each currency's own precision: 0, 2 and 3 decimals.
export const WithBalances: Story = {
    args: { showBalance: true },
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await expect(await screen.findByRole('option', { name: /Yen wallet.*15,000 ¥/ })).toBeVisible()
        await expect(screen.getByRole('option', { name: /Kuwait savings.*12\.345 KD/ })).toBeVisible()
    },
}

export const WithExcludedAccount: Story = {
    args: { excludeId: 2 },
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await waitFor(() => expect(screen.getAllByRole('option')).toHaveLength(3))
        await expect(screen.queryByRole('option', { name: /Euro card/ })).toBeNull()
    },
}

// The chosen account stays listed even when it is the excluded one.
export const SelectedStaysListed: Story = {
    args: { excludeId: 2, value: 2 },
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await expect(await screen.findByRole('option', { name: /Euro card/ })).toBeVisible()
    },
}

export const WithCreateOption: Story = {
    args: { allowCreate: true },
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await waitFor(() => expect(screen.getAllByRole('option')).toHaveLength(5))
    },
}

export const Picked: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await userEvent.click(await screen.findByRole('option', { name: /Euro card/ }))
        // Choosing remounts the select, so the trigger is looked up again.
        await expect(within(canvasElement).getByRole('combobox')).toHaveTextContent('Euro card')
    },
}
