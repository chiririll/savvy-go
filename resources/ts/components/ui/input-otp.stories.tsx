import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import { InputOTP, InputOTPGroup, InputOTPSlot } from './input-otp'

type OtpProps = { maxLength: number; 'aria-label': string; defaultValue?: string; disabled?: boolean }

const Otp = (props: OtpProps) => (
    <InputOTP {...props}>
        <InputOTPGroup>
            {[0, 1, 2, 3].map((index) => <InputOTPSlot key={index} index={index} />)}
        </InputOTPGroup>
    </InputOTP>
)

const meta = {
    title: 'ui/InputOTP',
    component: Otp,
    args: { maxLength: 4, 'aria-label': 'Code' },
} satisfies Meta<typeof Otp>

export default meta
type Story = StoryObj<typeof meta>

export const Empty: Story = {}
export const Partial: Story = { args: { defaultValue: '12' } }
export const Complete: Story = { args: { defaultValue: '1234' } }
export const Disabled: Story = { args: { disabled: true, defaultValue: '12' } }

export const NeverLongerThanItsLength: Story = {
    play: async ({ canvasElement }) => {
        const input = within(canvasElement).getByRole('textbox')
        await userEvent.type(input, '987654')
        await expect((input as HTMLInputElement).value).toHaveLength(4)
    },
}
