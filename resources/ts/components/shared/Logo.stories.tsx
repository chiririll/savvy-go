import type { Meta, StoryObj } from '@storybook/react-vite'

import { Logo } from './Logo'

const meta = { title: 'shared/Logo', component: Logo, args: { className: 'size-16', role: 'img', 'aria-label': 'Savvy' } } satisfies Meta<typeof Logo>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}
export const Small: Story = { args: { className: 'size-6' } }
export const Colored: Story = { args: { className: 'size-16 text-primary' } }
