import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'

import type { DraftContent } from './draft'
import { TagsStep } from './TagsStep'

const seeded: DraftContent = { baseCurrency: 'EUR', currencies: [], accounts: [], categories: [], tags: ['holiday'] }

const changed = fn()

const Harness = () => {
    const [content, setContent] = useState(seeded)
    return (
        <TagsStep
            content={content}
            onChange={(patch) => {
                setContent((c) => ({ ...c, ...patch }))
                changed(patch)
            }}
        />
    )
}

const meta = {
    title: 'features/spaces/create/TagsStep',
    parameters: { layout: 'padded', a11y: overlayA11y },
    beforeEach: () => changed.mockClear(),
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

// A tag is added once, whatever its case.
export const TagAdded: Story = {
    render: () => <Harness />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const field = canvas.getByRole('textbox', { name: /new tag/i })
        await userEvent.type(field, 'HOLIDAY{Enter}')
        await expect(changed).not.toHaveBeenCalled()
        await expect(field).toHaveValue('')
        await userEvent.type(field, 'work{Enter}')
        await expect(changed).toHaveBeenLastCalledWith({ tags: ['holiday', 'work'] })
        await userEvent.click(canvas.getByRole('button', { name: /remove holiday/i }))
        await expect(changed).toHaveBeenLastCalledWith({ tags: ['work'] })
    },
}
