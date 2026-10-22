import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { reply } from '@/test/api-mocks'
import type { PublicProvider } from '@/types/sso'

import { BrandTile } from './BrandIcon'
import { LoginPreviewCard } from './LoginPreviewCard'
import { SsoButtons } from './SsoButtons'

const providers: PublicProvider[] = [
    { slug: 'github', name: 'GitHub', preset: 'github', protocol: 'oauth2' },
    { slug: 'entra', name: 'Microsoft', preset: 'entra', protocol: 'oidc' },
    { slug: 'corp', name: 'Corporate', preset: 'custom_saml', protocol: 'saml' },
]

// The sign-in buttons, the preview of one in the provider form, and the brand tiles they use.
const meta = {
    title: 'features/sso/SsoButtons',
    component: SsoButtons,
    parameters: { layout: 'padded' },
    beforeEach: ({ msw }) => msw.use(reply('get', '/auth/sso/providers', providers)),
    decorators: [(Story) => <div className="max-w-sm rounded-xl border bg-card p-6">{Story()}</div>],
} satisfies Meta<typeof SsoButtons>

export default meta
type Story = StoryObj<typeof meta>

export const Buttons: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('button', { name: /continue with github/i })).toBeVisible()
        await expect(canvas.getAllByRole('button')).toHaveLength(3)
        await expect(canvas.getByText('or')).toBeVisible()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
export const InDarkTheme: Story = { globals: { theme: 'dark' } }

export const WithoutDivider: Story = {
    args: { showDivider: false },
    play: async ({ canvasElement }) => {
        await within(canvasElement).findByRole('button', { name: /github/i })
        await expect(within(canvasElement).queryByText('or')).toBeNull()
    },
}

// No providers: nothing, not even the divider.
export const NoProviders: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/auth/sso/providers', [])),
    play: async ({ canvasElement }) => {
        await new Promise((resolve) => setTimeout(resolve, 200))
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
        await expect(within(canvasElement).queryByText('or')).toBeNull()
    },
}

export const Preview: Story = {
    render: () => <LoginPreviewCard preset="google" name="Google Workspace" enabled />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/continue with google workspace/i)).toBeVisible()
        await expect(canvas.getByText(/visible on the sign-in screen/i)).toBeVisible()
    },
}

export const PreviewWithoutAName: Story = {
    render: () => <LoginPreviewCard preset="custom_oidc" name="" enabled={false} />,
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/hidden\. enable it/i)).toBeVisible()
    },
}

const BRANDS = ['github', 'google', 'entra', 'okta', 'gitlab', 'keycloak', 'authentik', 'custom_oidc', 'custom_saml', 'x']

export const Brands: Story = {
    render: () => (
        <div className="flex flex-wrap gap-3">
            {BRANDS.map((preset) => (
                <BrandTile key={preset} preset={preset} className="size-12" iconClassName="size-6" />
            ))}
        </div>
    ),
}

export const BrandsInDarkTheme: Story = { ...Brands, globals: { theme: 'dark' } }
