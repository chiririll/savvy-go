import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import { SpacePicker, type SpaceChoice } from './SpacePicker'

const spaces = [{ id: 1, name: 'Home' }, { id: 2, name: 'Trip' }, { id: 3, name: 'Business' }]

type Props = { initial?: SpaceChoice; spaces?: typeof spaces }

function Stateful({ initial = { all: false, ids: [1] }, spaces: list = spaces }: Props) {
    const [value, setValue] = useState(initial)
    return (
        <div className="max-w-md">
            <SpacePicker idPrefix="story" spaces={list} value={value} onChange={setValue} />
        </div>
    )
}

const meta = {
    title: 'features/api-tokens/SpacePicker',
    component: Stateful,
} satisfies Meta<typeof Stateful>

export default meta
type Story = StoryObj<typeof meta>

export const SomeSpaces: Story = {
    play: async ({ canvasElement }) => {
        const picker = within(canvasElement)
        await expect(picker.getByRole('checkbox', { name: 'Home' })).toBeChecked()
        await expect(picker.getByRole('checkbox', { name: 'Trip' })).not.toBeChecked()
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }
export const NoneChosen: Story = { args: { initial: { all: false, ids: [] } } }
export const NoSpaces: Story = { args: { spaces: [] } }
export const ManySpaces: Story = {
    args: { spaces: Array.from({ length: 15 }, (_, i) => ({ id: i + 1, name: `Space number ${i + 1}` })) },
}
export const LongName: Story = {
    args: { spaces: [{ id: 1, name: 'A space name far too long to fit on one line of a narrow dialog' }] },
}

export const AllSpaces: Story = {
    args: { initial: { all: true, ids: [] } },
    play: async ({ canvasElement }) => {
        const picker = within(canvasElement)
        await expect(picker.getByRole('checkbox', { name: 'Trip' })).toBeChecked()
        await expect(picker.getByRole('checkbox', { name: 'Trip' })).toBeDisabled()
    },
}

export const TickingASpace: Story = {
    play: async ({ canvasElement }) => {
        const trip = within(canvasElement).getByRole('checkbox', { name: 'Trip' })
        await userEvent.click(trip)
        await expect(trip).toBeChecked()
        await userEvent.click(within(canvasElement).getByRole('checkbox', { name: 'Home' }))
        await expect(within(canvasElement).getByRole('checkbox', { name: 'Home' })).not.toBeChecked()
    },
}

export const TurningAllOnClearsTheTicks: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('switch'))
        await expect(within(canvasElement).getByRole('checkbox', { name: 'Home' })).toBeDisabled()
        await userEvent.click(within(canvasElement).getByRole('switch'))
        await expect(within(canvasElement).getByRole('checkbox', { name: 'Home' })).not.toBeChecked()
    },
}
