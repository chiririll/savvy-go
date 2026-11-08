import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { Markdown } from './Markdown'

const meta = {
    title: 'shared/markdown/Markdown',
    component: Markdown,
} satisfies Meta<typeof Markdown>

export default meta
type Story = StoryObj<typeof meta>

export const ProductList: Story = {
    args: { children: '- milk\n- bread\n- **eggs**, 10 pcs' },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('milk')).toBeVisible()
    },
}

export const TaskList: Story = {
    args: { children: '- [x] milk\n- [ ] bread' },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findAllByRole('checkbox')).toHaveLength(2)
    },
}

export const Formatting: Story = {
    args: { children: 'Paid **in cash**, *no receipt*. See `#1042` and [the shop](https://example.com).' },
    play: async ({ canvasElement }) => {
        const link = await within(canvasElement).findByRole('link', { name: 'the shop' })
        await expect(link).toHaveAttribute('rel', 'noopener noreferrer')
    },
}

export const UnsafeContent: Story = {
    args: { children: '[click](javascript:alert(1))\n\n<script>alert(1)</script>\n\n![img](https://example.com/a.png)\n\n# Heading' },
    play: async ({ canvasElement }) => {
        await within(canvasElement).findByText('click')
        await expect(within(canvasElement).queryByRole('link')).toBeNull()
        await expect(canvasElement.querySelector('img, script, h1')).toBeNull()
    },
}
