import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import { tags } from '@/test/fixtures'
import { hang, reply } from '@/test/api-mocks'
import { withForm } from '@/test/form-decorator'

import { TagSelect } from './TagSelect'

const meta = {
    title: 'shared/TagSelect',
    component: TagSelect,
    args: { onChange: () => {} },
    render: (args) => {
        const [value, setValue] = useState(args.value ?? [])
        return <TagSelect {...args} value={value} onChange={setValue} />
    },
    beforeEach: ({ msw }) => msw.use(reply('get', '/tags', tags)),
} satisfies Meta<typeof TagSelect>

export default meta
type Story = StoryObj<typeof meta>

export const NoneSelected: Story = {}
export const SomeSelected: Story = { args: { value: [1, 3] } }
export const Bordered: Story = { args: { bordered: true } }

export const NoTags: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/tags', [])),
    play: async ({ canvasElement }) => {
        await new Promise((resolve) => setTimeout(resolve, 200))
        await expect(canvasElement.querySelector('[data-slot="badge"]')).toBeNull()
    },
}

export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/tags')) }

export const AsFormItem: Story = {
    args: { asFormItem: true, description: 'Pick any that fit.' },
    decorators: [withForm()],
}

export const ToggledByClick: Story = {
    play: async ({ canvasElement }) => {
        const work = await within(canvasElement).findByRole('button', { name: '#work' })
        await userEvent.click(work)
        await expect(work).toHaveAttribute('aria-pressed', 'true')
    },
}

// A chip is a button, so the keyboard reaches it.
export const ToggledByKeyboard: Story = {
    play: async ({ canvasElement }) => {
        const chip = await within(canvasElement).findByRole('button', { name: '#holiday' })
        chip.focus()
        await userEvent.keyboard('{Enter}')
        await expect(chip).toHaveAttribute('aria-pressed', 'true')
        await userEvent.keyboard(' ')
        await expect(chip).toHaveAttribute('aria-pressed', 'false')
    },
}
