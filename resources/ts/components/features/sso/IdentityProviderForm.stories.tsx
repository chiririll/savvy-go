import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { reply, settingsApi } from '@/test/api-mocks'

import { presets } from './fixtures'
import { IdentityProviderForm } from './IdentityProviderForm'

const meta = {
    title: 'features/sso/IdentityProviderForm',
    component: IdentityProviderForm,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => msw.use(reply('get', '/auth/sso/presets', presets), ...settingsApi()),
    args: { onSubmit: fn() },
} satisfies Meta<typeof IdentityProviderForm>

export default meta
type Story = StoryObj<typeof meta>

const submit = (canvasElement: HTMLElement) =>
    userEvent.click(within(canvasElement).getByRole('button', { name: /^save$/i }))

export const Empty: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/no rules yet/i)).toBeVisible()
        await expect(canvas.queryByText(/register these urls/i)).toBeNull()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
export const InDarkTheme: Story = { globals: { theme: 'dark' } }

export const Required: Story = {
    play: async ({ canvasElement, args }) => {
        await submit(canvasElement)
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Pick a provider type')).toBeVisible()
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

export const SlugMustBeLowercase: Story = {
    args: { defaultValues: { name: 'GitHub', slug: 'My GitHub', preset: 'github' } },
    play: async ({ canvasElement, args }) => {
        await submit(canvasElement)
        await waitFor(() =>
            expect(within(canvasElement).getByLabelText(/slug/i)).toHaveAttribute('aria-invalid', 'true'),
        )
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

// Picking a preset shows its own fields, and the URLs to register at the provider.
export const PresetPicked: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.type(canvas.getByLabelText(/slug/i), 'sso')
        await userEvent.click(await canvas.findByRole('combobox', { name: /provider type/i }))
        await userEvent.click(await screen.findByRole('option', { name: /keycloak/i }))
        await expect(await canvas.findByText('Usually master')).toBeVisible()
        await expect(canvas.getByText('https://savvy.example.com/api/auth/sso/sso/callback')).toBeVisible()
    },
}

export const Saml: Story = {
    args: { defaultValues: { name: 'Corporate', slug: 'corp', preset: 'custom_saml' }, presetLocked: true },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('https://savvy.example.com/api/auth/sso/corp/acs')).toBeVisible()
        await expect(canvas.getByText('https://savvy.example.com/api/auth/sso/corp/metadata')).toBeVisible()
        await expect(canvas.queryByRole('combobox', { name: /provider type/i })).toBeNull()
    },
}

// The server builds redirect URIs from the public URL, so without one there is nothing to register yet.
export const WithoutAppUrl: Story = {
    args: { defaultValues: { name: 'GitHub', slug: 'github', preset: 'github' } },
    beforeEach: ({ msw }) => msw.use(...settingsApi({ app_url: '' })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/set the public url/i)).toBeVisible()
    },
}

export const UrlCopied: Story = {
    args: { defaultValues: { name: 'GitHub', slug: 'github', preset: 'github' } },
    beforeEach: () => {
        Object.defineProperty(navigator, 'clipboard', { value: { writeText: fn(async () => {}) }, configurable: true })
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /^copy$/i }))
        await expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
            'https://savvy.example.com/api/auth/sso/github/callback',
        )
        await expect(await screen.findByText(/copied/i)).toBeVisible()
    },
}

// Editing keeps the slug and preset, and an empty secret keeps the stored one.
export const Edit: Story = {
    args: {
        isEdit: true,
        defaultValues: { name: 'Company GitHub', slug: 'github', preset: 'github', fields: { client_id: 'abc' } },
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByLabelText(/slug/i)).toBeDisabled()
        await expect(await canvas.findByRole('combobox', { name: /provider type/i })).toBeDisabled()
        await expect(await canvas.findByLabelText(/client secret/i)).toHaveAttribute('type', 'password')
    },
}

export const Submitted: Story = {
    args: { defaultValues: { name: 'GitHub', slug: 'github', preset: 'github' } },
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        await userEvent.type(await canvas.findByLabelText(/client id/i), 'abc')
        await userEvent.click(canvas.getByRole('switch', { name: 'Link by email' }))
        await submit(canvasElement)
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalled())
        await expect(firstSubmit(args.onSubmit)).toMatchObject({
            name: 'GitHub',
            slug: 'github',
            preset: 'github',
            fields: { client_id: 'abc' },
            link_by_email: false,
            default_role: 'user',
            role_mapping: [],
        })
    },
}

export const RuleAddedAndRemoved: Story = {
    args: { defaultValues: { name: 'GitHub', slug: 'github', preset: 'github' } },
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('button', { name: /add rule/i }))
        await expect(canvas.getByRole('textbox', { name: /claim/i })).toHaveValue('groups')
        await userEvent.type(canvas.getByRole('textbox', { name: /^value$/i }), 'finance-admins')
        await userEvent.click(canvas.getAllByRole('combobox', { name: /^role$/i })[0])
        await userEvent.click(await screen.findByRole('option', { name: /admin/i }))
        await submit(canvasElement)
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalled())
        await expect(firstSubmit(args.onSubmit).role_mapping).toEqual([
            { claim: 'groups', operator: 'contains', value: 'finance-admins', role: 'admin' },
        ])
        await userEvent.click(canvas.getByRole('button', { name: /remove rule/i }))
        await expect(canvas.getByText(/no rules yet/i)).toBeVisible()
    },
}

export const RuleNeedsAValue: Story = {
    args: { defaultValues: { name: 'GitHub', slug: 'github', preset: 'github' } },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /add rule/i }))
        await submit(canvasElement)
        await new Promise((resolve) => setTimeout(resolve, 200))
        await expect(args.onSubmit).not.toHaveBeenCalled()
    },
}

export const Submitting: Story = {
    args: { isSubmitting: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('button', { name: /saving/i })).toBeDisabled()
    },
}

// The login button preview follows the name and the switch.
export const WithPreview: Story = {
    args: { previewPreset: 'github', presetLocked: true, defaultValues: { preset: 'github' } },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.type(canvas.getByLabelText(/display name/i), 'Company GitHub')
        await expect(canvas.getByText(/continue with company github/i)).toBeVisible()
        await userEvent.click(canvas.getByRole('switch', { name: 'Enabled' }))
        await expect(canvas.getByText(/hidden\. enable it/i)).toBeVisible()
    },
}

/** What the form gave its first submit. */
const firstSubmit = (spy: unknown) =>
    (spy as { mock: { calls: [Record<string, unknown>][] } }).mock.calls[0][0]
