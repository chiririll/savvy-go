import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { hang, reply } from '@/test/api-mocks'

import { presets } from './fixtures'
import { PresetChooser } from './PresetChooser'
import { SelectedPresetBanner } from './SelectedPresetBanner'

const meta = {
    title: 'features/sso/PresetChooser',
    component: PresetChooser,
    parameters: { layout: 'padded' },
    beforeEach: ({ msw }) => msw.use(reply('get', '/auth/sso/presets', presets)),
    args: { onSelect: fn() },
} satisfies Meta<typeof PresetChooser>

export default meta
type Story = StoryObj<typeof meta>

// Built-in presets come first; the custom ones are under "bring your own".
export const Presets: Story = {
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        const github = await canvas.findByRole('button', { name: /github/i })
        await expect(canvas.getAllByRole('button')).toHaveLength(4)
        await userEvent.click(github)
        await expect(args.onSelect).toHaveBeenCalledWith('github')
    },
}

export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/auth/sso/presets')) }

export const CustomPicked: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /saml/i }))
        await expect(args.onSelect).toHaveBeenCalledWith('custom_saml')
    },
}

// The banner above the form once a preset is picked.
export const Banner: Story = {
    render: () => <SelectedPresetBanner preset="keycloak" onChange={fn()} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Keycloak')).toBeVisible()
        await expect(canvas.getByRole('button', { name: /change/i })).toBeVisible()
    },
}

export const BannerWithoutChange: Story = {
    render: () => <SelectedPresetBanner preset="github" />,
    play: async ({ canvasElement }) => {
        await within(canvasElement).findByText('GitHub')
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

// A preset the catalog does not have shows its key.
export const BannerUnknownPreset: Story = {
    render: () => <SelectedPresetBanner preset="ping" />,
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('ping')).toBeVisible()
    },
}
