import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ListPage } from '@/components/shared'
import { createUserColumns, UserFormDialog, UserPasswordLinkDialog } from '@/components/features/users'
import { useUsers, useDeleteUser, useCreateUser, useUpdateUser, useIssuePasswordToken, useResourceFormDialog } from '@/hooks'
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
    const isReadOnly = useReadOnly()

    const [linkUrl, setLinkUrl] = useState<string | null>(null)
    const form = useResourceFormDialog<User, CreateUserFormData | UpdateUserFormData>({
        items: users ?? [],
        isLoading,
        syncSearchParams: false,
    })

    const showLink = (token?: string) => {
        if (!token) return
        form.setOpen(false)
        setLinkUrl(setPasswordUrl(token))
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
        createUser.mutate(
            {
                name: createData.name,
                email: createData.email,
                role: createData.role,
                password: createData.setPassword ? createData.password : undefined,
            },
            {
                onSuccess: (user) => {
                    if (user.token) {
                        showLink(user.token)
                    } else {
                        form.setOpen(false)
                    }
                },
            },
        )
    }

    const handleReset = (user: User) => {
        issueToken.mutate(user.id, {
            onSuccess: (result) => showLink(result.token),
        })
    }

    const columns = createUserColumns(
        (id) => deleteUser.mutate(id),
        form.openEdit,
        handleReset,
        currentUser?.id,
        isReadOnly
    )

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
            />

            <UserPasswordLinkDialog
                url={linkUrl}
                open={linkUrl !== null}
                onOpenChange={(open) => {
                    if (!open) setLinkUrl(null)
                }}
            />
        </>
    )
}
