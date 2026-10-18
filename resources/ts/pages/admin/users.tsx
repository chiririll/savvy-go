import { useState } from 'react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { ListPage } from '@/components/shared'
import { createUserColumns, UserFormDialog, UserPasswordLinkDialog } from '@/components/features/users'
import {
    useUsers,
    useDeleteUser,
    useCreateUser,
    useUpdateUser,
    useIssuePasswordToken,
    useResourceFormDialog,
    useSetEmailVerified,
    useMailStatus,
    useSettings,
} from '@/hooks'
import { useUser as useCurrentUser } from '@/stores/auth'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import { User } from '@/types/users'
import { CreateUserFormData, UpdateUserFormData } from '@/schemas/users'

function setPasswordUrl(token: string): string {
    return `${window.location.origin}/set-password/${token}`
}

export default function UsersPage() {
    const { t } = useTranslation('pages')
    const currentUser = useCurrentUser()
    const { data: users, isLoading } = useUsers()
    const deleteUser = useDeleteUser()
    const createUser = useCreateUser()
    const updateUser = useUpdateUser()
    const issueToken = useIssuePasswordToken()
    const setEmailVerified = useSetEmailVerified()
    const { data: mail } = useMailStatus()
    const { data: settings } = useSettings()
    const isReadOnly = useReadOnly()

    // Emailing needs a working mail server, the public URL for the link, and password sign-in.
    const canEmail = !!mail?.enabled && mail.appUrlSet && settings?.password_login_enabled !== false
    // With verification on, a link the admin hands over proves nothing about the mailbox: only email will do.
    const mustEmail = !!settings?.require_email_verification

    const [link, setLink] = useState<{ url: string; expiresAt?: string } | null>(null)
    const form = useResourceFormDialog<User, CreateUserFormData | UpdateUserFormData>({
        items: users ?? [],
        isLoading,
        syncSearchParams: false,
    })

    const showLink = (token?: string, expiresAt?: string) => {
        if (!token) return
        form.setOpen(false)
        setLink({ url: setPasswordUrl(token), expiresAt })
    }

    const handleSubmit = (data: CreateUserFormData | UpdateUserFormData) => {
        if (form.entity) {
            const { password, ...rest } = data
            updateUser.mutate(
                { id: form.entity.id, data: { ...rest, password: password || undefined } },
                { onSuccess: () => form.setOpen(false) },
            )
            return
        }

        const createData = data as CreateUserFormData
        const send = !createData.setPassword && canEmail && (mustEmail || !!createData.sendEmail)
        createUser.mutate(
            {
                name: createData.name,
                email: createData.email,
                role: createData.role,
                password: createData.setPassword ? createData.password : undefined,
                send: send || undefined,
            },
            {
                onSuccess: (user) => {
                    if (user.token) {
                        showLink(user.token, user.expiresAt)
                    } else if (user.emailError) {
                        form.setOpen(false)
                        toast.error(t('users.linkEmailFailed', { email: user.email }))
                    } else {
                        if (user.emailSent) toast.success(t('users.linkEmailed', { email: user.email }))
                        form.setOpen(false)
                    }
                },
            },
        )
    }

    const handleReset = (user: User) => {
        issueToken.mutate(
            { id: user.id },
            {
                onSuccess: (result) => showLink(result.token, result.expiresAt),
            },
        )
    }

    const handleEmailReset = (user: User) => issueToken.mutate({ id: user.id, send: true })

    const columns = createUserColumns({
        onDelete: (id) => deleteUser.mutate(id),
        onEdit: form.openEdit,
        onResetPassword: handleReset,
        currentUserId: currentUser?.id,
        isReadOnly,
        onEmailReset: canEmail ? handleEmailReset : undefined,
        showUnverified: !!mail?.enabled || mustEmail,
        onSetEmailVerified: (user, verified) => setEmailVerified.mutate({ id: user.id, verified }),
    })

    return (
        <>
            <ListPage
                title={t('users.title')}
                description={t('users.description')}
                createLabel={t('users.create')}
                onCreateClick={isReadOnly ? undefined : form.openCreate}
                data={users ?? []}
                columns={columns}
                isLoading={isLoading}
            />

            <UserFormDialog
                user={form.entity}
                open={form.open}
                onOpenChange={form.setOpen}
                onSubmit={handleSubmit}
                isSubmitting={createUser.isPending || updateUser.isPending}
                mail={{ canEmail, mustEmail }}
            />

            <UserPasswordLinkDialog
                url={link?.url ?? null}
                expiresAt={link?.expiresAt}
                open={link !== null}
                onOpenChange={(open) => {
                    if (!open) setLink(null)
                }}
            />
        </>
    )
}
