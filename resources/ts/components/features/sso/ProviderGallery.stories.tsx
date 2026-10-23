import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'

import { provider } from './fixtures'
import { ProviderCard, ProviderGridSkeleton, ProvidersEmptyState } from './ProviderGallery'

const onDelete = fn()
const onTest = fn()

// The cards of the identity provider page, its empty state and its loading grid.
const meta = {
    title: 'features/sso/ProviderGallery',
    parameters: { layout: 'padded', a11y: overlayA11y },
    beforeEach: () => {
        onDelete.mockClear()
        onTest.mockClear()
    },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

const grid = (...cards: Parameters<typeof provider>[0][]) => (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {cards.map((overrides, index) => (
            <ProviderCard
                key={index}
                provider={provider({ id: index + 1, ...overrides })}
                index={index}
                onDelete={onDelete}
                onTest={onTest}
            />
        ))}
    </div>
)

export const Cards: Story = {
    render: () =>
        grid(
            {},
            { name: 'Keycloak', slug: 'keycloak', preset: 'keycloak', protocol: 'oidc' },
            { name: 'Corporate SAML', slug: 'corp', preset: 'custom_saml', protocol: 'saml', enabled: false },
        ),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByRole('link', { name: 'Edit Company GitHub' })).toHaveAttribute(
            'href',
            '/admin/providers/1/edit',
        )
        await expect(canvas.getAllByText('Active')).toHaveLength(2)
        await expect(canvas.getByText('Disabled')).toBeVisible()
    },
}

export const CardsInRussian: Story = { ...Cards, play: undefined, globals: { locale: 'ru' } }
export const LongName: Story = {
    render: () => grid({ name: 'A provider with a very long name for the whole company', slug: 'a-very-long-slug' }),
}

export const Tested: Story = {
    render: () => grid({}),
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /more actions/i }))
        await userEvent.click(await screen.findByRole('menuitem', { name: /test connection/i }))
        await expect(onTest).toHaveBeenCalledWith(1)
    },
}

export const DeleteAsksFirst: Story = {
    render: () => grid({}),
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /more actions/i }))
        await userEvent.click(await screen.findByRole('menuitem', { name: /delete/i }))
        const dialog = within(await screen.findByRole('alertdialog'))
        await expect(dialog.getByText(/"Company GitHub" will be permanently removed/)).toBeVisible()
        await expect(onDelete).not.toHaveBeenCalled()
        await userEvent.click(dialog.getByRole('button', { name: /delete/i }))
        await expect(onDelete).toHaveBeenCalledWith(1)
    },
}

export const Empty: Story = {
    render: () => <ProvidersEmptyState />,
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('link', { name: /add provider/i })).toHaveAttribute(
            'href',
            '/admin/providers/create',
        )
    },
}

export const EmptyInRussian: Story = { ...Empty, play: undefined, globals: { locale: 'ru' } }
export const Loading: Story = { render: () => <ProviderGridSkeleton /> }
