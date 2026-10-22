import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { Step, Stepper } from './stepper'

const meta = {
    title: 'ui/Stepper',
    component: Stepper,
    args: { activeStep: 1, onStepChange: fn(), children: null },
    render: (args) => (
        <Stepper {...args}>
            <Step>Basics</Step>
            <Step>Currencies</Step>
            <Step>Members</Step>
        </Stepper>
    ),
} satisfies Meta<typeof Stepper>

export default meta
type Story = StoryObj<typeof meta>

export const Middle: Story = {}
export const First: Story = { args: { activeStep: 0 } }
export const Last: Story = { args: { activeStep: 2 } }
export const Vertical: Story = { args: { orientation: 'vertical' } }

export const GoBack: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByText('Basics'))
        await expect(args.onStepChange).toHaveBeenCalledWith(0)
    },
}

export const CannotJumpAhead: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByText('Members'))
        await expect(args.onStepChange).not.toHaveBeenCalled()
    },
}
