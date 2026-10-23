import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { space } from '@/test/app-state'

import { ConfirmStep } from './ConfirmStep'
import { CopyStep } from './CopyStep'
import { noCopy, type CopyKinds } from './draft'
import { NameStep } from './NameStep'

// The small steps of the create-space wizard: name, copy source and the summary.
const meta = { title: 'features/spaces/create/Steps', parameters: { layout: 'padded' } } satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

const onSubmit = fn()
const onCopy = fn()

const Name = ({ initial = '' }: { initial?: string }) => {
    const [name, setName] = useState(initial)
    const [look, setLook] = useState({ emoji: '', color: '' })
    return <NameStep name={name} onNameChange={setName} {...look} onLookChange={setLook} onSubmit={onSubmit} />
}

export const NameEmpty: Story = {
    render: () => <Name />,
    beforeEach: () => onSubmit.mockClear(),
    play: async ({ canvasElement }) => {
        const field = within(canvasElement).getByLabelText('Name')
        await expect(field).toHaveFocus()
        // Enter does nothing until there is a name.
        await userEvent.type(field, '   {Enter}')
        await expect(onSubmit).not.toHaveBeenCalled()
        await userEvent.type(field, 'Family{Enter}')
        await expect(onSubmit).toHaveBeenCalledTimes(1)
    },
}

const Copy = ({ spaces = [space(), { ...space(), id: 2, name: 'Work' }], from = null as number | null }) => {
    const [sourceId, setSourceId] = useState(from)
    const [copy, setCopy] = useState<CopyKinds>(noCopy())
    return (
        <CopyStep
            spaces={spaces}
            sourceId={sourceId}
            copy={copy}
            onSourceChange={setSourceId}
            onCopyChange={(next) => {
                setCopy(next)
                onCopy(next)
            }}
        />
    )
}

export const CopyFromDefaults: Story = {
    render: () => <Copy />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByRole('combobox', { name: /start from/i })).toHaveTextContent(/the defaults/i)
        await expect(canvas.queryByRole('checkbox')).toBeNull()
    },
}

export const CopyFromASpace: Story = {
    render: () => <Copy />,
    parameters: { a11y: overlayA11y },
    beforeEach: () => onCopy.mockClear(),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('combobox', { name: /start from/i }))
        await userEvent.click(await screen.findByRole('option', { name: 'Work' }))
        await expect(await canvas.findAllByRole('checkbox')).toHaveLength(4)
        await userEvent.click(canvas.getByRole('checkbox', { name: /accounts \(without balances\)/i }))
        await expect(onCopy).toHaveBeenLastCalledWith({ ...noCopy(), accounts: true })
        await expect(canvas.getByText(/one-time copy/i)).toBeVisible()
    },
}

export const Confirm: Story = {
    render: () => (
        <ConfirmStep
            request={{
                name: 'Family',
                baseCurrency: 'EUR',
                currencies: ['JPY', 'KWD'],
                accounts: [
                    { name: 'Cash', currency: 'EUR' },
                    { name: 'Yen wallet', currency: 'JPY', initialBalance: '15000' },
                ],
                categories: [],
                tags: ['holiday'],
            }}
        />
    ),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('JPY, KWD')).toBeVisible()
        await expect(canvas.getByText('2')).toBeVisible()
    },
}

export const ConfirmWithoutOtherCurrencies: Story = {
    render: () => <ConfirmStep request={{ name: 'Solo', baseCurrency: 'USD', currencies: [], accounts: [] }} />,
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('—')).toBeVisible()
    },
}
