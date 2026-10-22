import type { Meta, StoryObj } from '@storybook/react-vite'

import { Avatar, AvatarFallback, AvatarImage } from './avatar'

const meta = { title: 'ui/Avatar', component: Avatar } satisfies Meta<typeof Avatar>

export default meta
type Story = StoryObj<typeof meta>

const GREY_SQUARE =
    'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="40" height="40"><rect width="40" height="40" fill="%23888"/></svg>'

export const WithImage: Story = {
    render: (args) => (
        <Avatar {...args}>
            <AvatarImage src={GREY_SQUARE} alt="Kirill" />
            <AvatarFallback>KC</AvatarFallback>
        </Avatar>
    ),
}

export const Fallback: Story = {
    render: (args) => (
        <Avatar {...args}>
            <AvatarImage src="/broken.png" alt="" />
            <AvatarFallback>KC</AvatarFallback>
        </Avatar>
    ),
}
