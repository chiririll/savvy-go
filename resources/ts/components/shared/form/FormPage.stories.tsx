import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { FormPage } from './FormPage'

const meta = {
    title: 'shared/form/FormPage',
    component: FormPage,
    args: { title: 'Edit account', backLink: '/accounts', children: <p>The form goes here.</p> },
} satisfies Meta<typeof FormPage>

export default meta
type Story = StoryObj<typeof meta>

export const Loaded: Story = {}
export const Loading: Story = {
    args: { isLoading: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByText('The form goes here.')).toBeNull()
    },
}
export const WithoutBackLink: Story = { args: { backLink: undefined } }
export const SetsTheTabTitle: Story = {
    play: async () => {
        await expect(document.title).toContain('Edit account')
    },
}
