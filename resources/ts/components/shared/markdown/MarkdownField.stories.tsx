import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, userEvent, within } from 'storybook/test'

import { MarkdownField } from './MarkdownField'

function Controlled(props: Partial<React.ComponentProps<typeof MarkdownField>>) {
    const [value, setValue] = useState(props.value ?? '')
    return <MarkdownField {...props} value={value} onChange={setValue} />
}

const meta = {
    title: 'shared/markdown/MarkdownField',
    component: MarkdownField,
    render: (args) => <Controlled {...args} />,
    args: { value: '', onChange: () => {}, placeholder: 'Note' },
} satisfies Meta<typeof MarkdownField>

export default meta
type Story = StoryObj<typeof meta>

export const Empty: Story = {}

export const Filled: Story = { args: { value: '- milk\n- bread\n- [ ] eggs' } }

export const Compact: Story = { args: { rows: 2, value: 'one line' } }

export const Disabled: Story = { args: { disabled: true, value: '- locked' } }

export const Invalid: Story = { args: { 'aria-invalid': true, value: 'bad' } }

export const ToolbarBold: Story = {
    play: async ({ canvasElement }) => {
        const c = within(canvasElement)
        const box = c.getByPlaceholderText('Note')
        await userEvent.type(box, 'milk')
        await userEvent.keyboard('{Control>}a{/Control}')
        await userEvent.click(c.getByRole('button', { name: /bold|жирный/i }))
        await expect(box).toHaveValue('**milk**')
    },
}

export const EnterContinuesList: Story = {
    play: async ({ canvasElement }) => {
        const c = within(canvasElement)
        const box = c.getByPlaceholderText('Note')
        await userEvent.click(c.getByRole('button', { name: /^list$|^список$/i }))
        await userEvent.type(box, 'milk{Enter}bread{Enter}')
        await expect(box).toHaveValue('- milk\n- bread\n- ')
        await userEvent.keyboard('{Enter}')
        await expect(box).toHaveValue('- milk\n- bread\n')
    },
}

export const Preview: Story = {
    args: { value: '- milk\n- **bread**' },
    play: async ({ canvasElement }) => {
        const c = within(canvasElement)
        await userEvent.click(c.getByRole('button', { name: /preview|просмотр/i }))
        await expect(await c.findByText('bread')).toBeVisible()
        await expect(c.queryByPlaceholderText('Note')).toBeNull()
    },
}

export const PreviewEmpty: Story = {
    play: async ({ canvasElement }) => {
        const c = within(canvasElement)
        await userEvent.click(c.getByRole('button', { name: /preview|просмотр/i }))
        await expect(await c.findByText(/nothing to preview|нечего/i)).toBeVisible()
    },
}

// A cross at the right end of the toolbar, for a field that folds into a compact one.
export const WithClose: Story = {
    args: { value: '- milk', onClose: () => {} },
    play: async ({ canvasElement }) => {
        const c = within(canvasElement)
        await expect(c.getByRole('button', { name: /close the editor|закрыть редактор/i })).toBeVisible()
    },
}
