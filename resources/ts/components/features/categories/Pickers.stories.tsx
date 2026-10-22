import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, within } from 'storybook/test'

import { COLOR_OPTIONS } from '@/constants'
import { emojiPickerA11y } from '@/test/a11y'

import { CategoryPreview } from './CategoryPreview'
import { ColorPicker } from './ColorPicker'
import { IconPicker } from './IconPicker'
import { TypeSelector } from './TypeSelector'

// The small controls of the category form.
const meta = { title: 'features/categories/Pickers', parameters: { layout: 'padded' } } satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

export const Colors: Story = {
    render: () => {
        const [color, setColor] = useState(COLOR_OPTIONS[3])
        return <ColorPicker value={color} onChange={setColor} />
    },
    play: async ({ canvasElement }) => {
        const swatches = within(canvasElement).getAllByRole('button')
        await expect(swatches.filter((swatch) => swatch.getAttribute('aria-pressed') === 'true')).toHaveLength(1)
        await userEvent.click(swatches[0])
        await expect(swatches[0]).toHaveAttribute('aria-pressed', 'true')
    },
}

export const ColorsWithError: Story = {
    render: () => <ColorPicker value="" onChange={() => {}} error="Pick a colour" />,
}

export const ColorsInRussian: Story = { ...Colors, globals: { locale: 'ru' } }

export const Icons: Story = {
    render: () => {
        const [icon, setIcon] = useState('🛒')
        return <IconPicker value={icon} onChange={setIcon} />
    },
}

export const IconsOpened: Story = {
    ...Icons,
    parameters: { layout: 'padded', a11y: emojiPickerA11y },
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: 'Icon' }))
        await expect(await screen.findByRole('dialog')).toBeVisible()
    },
}

export const IconsWithError: Story = {
    render: () => <IconPicker value="" onChange={() => {}} error="Pick an icon" />,
}

export const IconsInDarkTheme: Story = { ...IconsOpened, globals: { theme: 'dark' } }

export const Types: Story = {
    render: () => {
        const [type, setType] = useState<'income' | 'expense'>('expense')
        return <TypeSelector value={type} onChange={setType} />
    },
}

export const TypesLocked: Story = {
    render: () => <TypeSelector value="expense" onChange={() => {}} disabled disabledHelp="The default category keeps its type." />,
}

export const TypesWithError: Story = {
    render: () => <TypeSelector value="expense" onChange={() => {}} error="Pick a type" />,
}

export const Preview: Story = {
    render: () => <CategoryPreview name="Groceries" icon="🛒" color="#16a34a" />,
}

export const PreviewWithoutName: Story = {
    render: () => <CategoryPreview name="" icon="🛒" color="#16a34a" />,
}
