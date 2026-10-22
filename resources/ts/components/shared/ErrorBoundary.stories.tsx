import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { ErrorBoundary } from './ErrorBoundary'

const Broken = () => {
    throw new Error('Rates table is missing')
}

const meta = {
    title: 'shared/ErrorBoundary',
    component: ErrorBoundary,
} satisfies Meta<typeof ErrorBoundary>

export default meta
type Story = StoryObj<typeof meta>

export const Healthy: Story = {
    args: { children: <p>All fine</p> },
}

export const Failed: Story = {
    args: { children: <Broken /> },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('Rates table is missing')).toBeVisible()
        await expect(within(canvasElement).getByRole('button')).toBeVisible()
    },
}

export const FailedWithFallback: Story = {
    args: { children: <Broken />, fallback: <p>Custom fallback</p> },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('Custom fallback')).toBeVisible()
    },
}

export const FailedInRussian: Story = { ...Failed, globals: { locale: 'ru' } }
