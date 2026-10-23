import { useState } from 'react'
import { useForm } from 'react-hook-form'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { Form } from '@/components/ui/form'

import { AdminFieldsStep } from './admin-setup/AdminFieldsStep'
import { BootstrapStep } from './admin-setup/BootstrapStep'
import { RestoreStep } from './admin-setup/RestoreStep'
import type { AdminSetupValues } from './admin-setup/useAdminSetup'
import { OnboardingLayout } from './OnboardingLayout'

const onNext = fn()
const onBack = fn()
const onSubmit = fn()

// The first-run screens: the bootstrap password, then the first admin or a restore, each in its card.
const meta = {
    title: 'features/onboarding/AdminSetup',
    parameters: { layout: 'fullscreen', app: { signedOut: true } },
    beforeEach: () => {
        for (const spy of [onNext, onBack, onSubmit]) spy.mockClear()
    },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

const Bootstrap = ({ environment, error = null }: { environment?: 'docker' | 'systemd' | 'binary'; error?: string | null }) => {
    const [password, setPassword] = useState('')
    return (
        <OnboardingLayout title="Set up Savvy" description="Prove you run this server.">
            <div className="space-y-4">
                <BootstrapStep
                    hint={environment ? { environment, file: '/data/bootstrap-password' } : null}
                    password={password}
                    onPasswordChange={setPassword}
                    error={error}
                    isLoading={false}
                    onNext={onNext}
                />
            </div>
        </OnboardingLayout>
    )
}

export const BootstrapInDocker: Story = {
    render: () => <Bootstrap environment="docker" />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('docker logs <container>')).toBeVisible()
        await expect(canvas.getByText('docker exec <container> cat /data/bootstrap-password')).toBeVisible()
    },
}

export const BootstrapUnderSystemd: Story = {
    render: () => <Bootstrap environment="systemd" />,
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('journalctl -u savvy-go')).toBeVisible()
    },
}

// A plain binary prints the password in its terminal: no commands to show.
export const BootstrapAsBinary: Story = {
    render: () => <Bootstrap environment="binary" />,
    play: async ({ canvasElement }) => {
        await expect(canvasElement.querySelector('code')).toBeNull()
    },
}

export const BootstrapWithoutHint: Story = { render: () => <Bootstrap /> }

export const BootstrapWrong: Story = {
    render: () => <Bootstrap environment="docker" error="The bootstrap password is wrong." />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByLabelText(/bootstrap password/i)).toHaveAttribute('aria-invalid', 'true')
        await expect(canvas.getByText('The bootstrap password is wrong.')).toBeVisible()
    },
}

export const BootstrapChoices: Story = {
    render: () => <Bootstrap environment="docker" />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.type(canvas.getByLabelText(/bootstrap password/i), 'ABCDE-12345')
        await userEvent.click(canvas.getByRole('button', { name: /create.*admin/i }))
        await expect(onNext).toHaveBeenLastCalledWith('admin')
        await userEvent.click(canvas.getByRole('button', { name: /restore/i }))
        await expect(onNext).toHaveBeenLastCalledWith('restore')
    },
}

const Admin = ({ isLoading = false }: { isLoading?: boolean }) => {
    const form = useForm<AdminSetupValues>({
        defaultValues: { name: '', email: '', password: '', password_confirmation: '' },
    })
    return (
        <OnboardingLayout title="Create the admin" description="The first account runs the server.">
            <Form {...form}>
                <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
                    <AdminFieldsStep form={form} isLoading={isLoading} onBack={onBack} />
                </form>
            </Form>
        </OnboardingLayout>
    )
}

export const AdminFields: Story = {
    render: () => <Admin />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.type(canvas.getByLabelText(/your name/i), 'Kirill')
        await userEvent.type(canvas.getByLabelText(/^email/i), 'kirill@example.com')
        await expect(canvas.getByLabelText(/^email/i)).toHaveAttribute('autocomplete', 'email')
        await userEvent.click(canvas.getByRole('button', { name: /back/i }))
        await expect(onBack).toHaveBeenCalled()
    },
}

export const AdminSaving: Story = {
    render: () => <Admin isLoading />,
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('button', { name: /back/i })).toBeDisabled()
    },
}

const Restore = ({ error = null }: { error?: string | null }) => {
    const [file, setFile] = useState<File | null>(null)
    const [appKey, setAppKey] = useState('')
    return (
        <OnboardingLayout title="Restore a backup" description="Bring a whole server back.">
            <form
                className="space-y-4"
                onSubmit={(e) => {
                    e.preventDefault()
                    onSubmit(file?.name, appKey)
                }}
            >
                <RestoreStep
                    backupFile={file}
                    onFileChange={setFile}
                    appKey={appKey}
                    onAppKeyChange={setAppKey}
                    error={error}
                    isLoading={false}
                    onBack={onBack}
                />
            </form>
        </OnboardingLayout>
    )
}

export const RestoreBackup: Story = {
    render: () => <Restore />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.upload(canvas.getByLabelText(/backup file/i), new File(['x'], 'server.zip'))
        // Only a Laravel database needs its old APP_KEY.
        await expect(canvas.queryByLabelText(/app key/i)).toBeNull()
        await userEvent.click(canvas.getByRole('button', { name: /restore/i }))
        await expect(onSubmit).toHaveBeenCalledWith('server.zip', '')
    },
}

export const RestoreLaravel: Story = {
    render: () => <Restore />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.upload(canvas.getByLabelText(/backup file/i), new File(['x'], 'database.sqlite'))
        await userEvent.type(await canvas.findByLabelText(/app key/i), 'base64:abc')
        await userEvent.click(canvas.getByRole('button', { name: /restore/i }))
        await expect(onSubmit).toHaveBeenCalledWith('database.sqlite', 'base64:abc')
    },
}

export const RestoreRefused: Story = {
    render: () => <Restore error="This is not a Savvy backup." />,
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByLabelText(/backup file/i)).toHaveAttribute('aria-invalid', 'true')
    },
}
