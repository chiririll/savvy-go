import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { fail, hang, reply, spy } from '@/test/api-mocks'
import { overlayA11y } from '@/test/a11y'
import type { MailStatus } from '@/types/mail'

import { MailSection } from './MailSection'

const mail: MailStatus = {
    enabled: true,
    problem: null,
    host: 'smtp.example.com',
    port: 587,
    protocol: 'starttls',
    user: 'savvy',
    from: 'Savvy <savvy@example.com>',
    skipVerify: false,
    timeout: 10,
    passwordSet: true,
    appUrlSet: true,
    writable: true,
}

const status = (overrides: Partial<MailStatus> = {}) => reply('get', '/admin/mail', { ...mail, ...overrides })

const meta = {
    title: 'features/mail/MailSection',
    component: MailSection,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => msw.use(status()),
} satisfies Meta<typeof MailSection>

export default meta
type Story = StoryObj<typeof meta>

export const Working: Story = {
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByDisplayValue('smtp.example.com')).toBeVisible()
        await expect(within(canvasElement).getByLabelText(/^test|send a test/i)).toBeVisible()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }

export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/admin/mail')) }

export const Off: Story = {
    beforeEach: ({ msw }) => msw.use(status({ enabled: false, host: '', user: '', from: '', passwordSet: false })),
    play: async ({ canvasElement }) => {
        await within(canvasElement).findByRole('switch')
        await expect(within(canvasElement).queryByLabelText(/host/i)).toBeNull()
    },
}

export const BrokenSavedConfig: Story = {
    beforeEach: ({ msw }) => msw.use(status({ enabled: false, problem: 'dial tcp: connection refused' })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/connection refused/i)).toBeVisible()
        await expect(within(canvasElement).getByDisplayValue('smtp.example.com')).toBeVisible()
    },
}

export const WithoutAppUrl: Story = {
    beforeEach: ({ msw }) => msw.use(status({ appUrlSet: false })),
}

export const FileNotWritable: Story = {
    beforeEach: ({ msw }) => msw.use(status({ writable: false })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByDisplayValue('smtp.example.com')).toBeDisabled()
        await expect(within(canvasElement).queryByRole('button', { name: /^save$/i })).toBeNull()
    },
}

export const PlainSmtpHasNoCertificateOption: Story = {
    beforeEach: ({ msw }) => msw.use(status({ protocol: 'smtp', port: 25 })),
    play: async ({ canvasElement }) => {
        await within(canvasElement).findByDisplayValue('smtp.example.com')
        await expect(within(canvasElement).getAllByRole('switch')).toHaveLength(1)
    },
}

export const NoPasswordStored: Story = {
    beforeEach: ({ msw }) => msw.use(status({ passwordSet: false, user: '' })),
}

export const StoredPasswordShownAsDots: Story = {
    play: async ({ canvasElement }) => {
        const password = await within(canvasElement).findByDisplayValue('••••••')
        await userEvent.click(password)
        await expect(password).toHaveValue('')
    },
}

export const ChoosingAProtocolFollowsThePort: Story = {
    play: async ({ canvasElement }) => {
        await within(canvasElement).findByDisplayValue('smtp.example.com')
        await userEvent.click(within(canvasElement).getByRole('combobox'))
        await userEvent.click(await screen.findByRole('option', { name: /smtps/i }))
        await waitFor(() => expect(within(canvasElement).getByDisplayValue('465')).toBeVisible())
    },
}

export const Saved: Story = {
    beforeEach: ({ msw }) => {
        const check = spy('post', '/admin/mail/check', { ok: true })
        const save = spy('put', '/admin/mail', mail)
        saved.check = check.bodies
        saved.save = save.bodies
        msw.use(check.handler, save.handler, status())
    },
    play: async ({ canvasElement }) => {
        await within(canvasElement).findByDisplayValue('smtp.example.com')
        await userEvent.click(within(canvasElement).getByRole('button', { name: /^save$/i }))
        await waitFor(() => expect(saved.save).toHaveLength(1))
        // The stored password is not sent back unless it was changed.
        await expect(saved.save[0]).not.toHaveProperty('password')
        await expect(saved.check).toHaveLength(1)
    },
}
const saved: { check: unknown[]; save: unknown[] } = { check: [], save: [] }

// A server that cannot be reached is refused before anything is saved.
export const NotSavedWhenTheCheckFails: Story = {
    beforeEach: ({ msw }) => {
        const save = spy('put', '/admin/mail', mail)
        refused.save = save.bodies
        msw.use(fail('post', '/admin/mail/check', 'Could not sign in', 422), save.handler, status())
    },
    play: async ({ canvasElement }) => {
        await within(canvasElement).findByDisplayValue('smtp.example.com')
        await userEvent.click(within(canvasElement).getByRole('button', { name: /^save$/i }))
        await new Promise((resolve) => setTimeout(resolve, 400))
        await expect(refused.save).toHaveLength(0)
    },
}
const refused: { save: unknown[] } = { save: [] }

export const ConnectionChecked: Story = {
    beforeEach: ({ msw }) => {
        const check = spy('post', '/admin/mail/check', { ok: true })
        checked.bodies = check.bodies
        msw.use(check.handler)
    },
    play: async ({ canvasElement }) => {
        await within(canvasElement).findByDisplayValue('smtp.example.com')
        await userEvent.click(within(canvasElement).getByRole('button', { name: /test connection/i }))
        await waitFor(() => expect(checked.bodies).toHaveLength(1))
        await expect(checked.bodies[0]).toMatchObject({ host: 'smtp.example.com', port: 587 })
    },
}
const checked: { bodies: unknown[] } = { bodies: [] }

export const TestMailSent: Story = {
    beforeEach: ({ msw }) => {
        const test = spy('post', '/admin/mail/test', { to: 'kirill@example.com' })
        sent.bodies = test.bodies
        msw.use(test.handler)
    },
    play: async ({ canvasElement }) => {
        const to = await within(canvasElement).findByLabelText(/test/i, { selector: 'input[type=email]' })
        await userEvent.type(to, 'me@example.com')
        await userEvent.click(within(canvasElement).getByRole('button', { name: /send/i }))
        await waitFor(() => expect(sent.bodies).toEqual([{ to: 'me@example.com' }]))
    },
}
const sent: { bodies: unknown[] } = { bodies: [] }
