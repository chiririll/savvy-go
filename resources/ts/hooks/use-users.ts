import { useQuery } from '@tanstack/react-query'
import { usersApi } from '@/api/users'
import type { UserFormData } from '@/schemas'
import { useResourceItem, useResourceMutation } from './use-crud'
import i18n from '@/lib/i18n'
import { useAuthStore } from '@/stores/auth'
import type { User } from '@/types/users'

const QUERY_KEY = ['users']

/** The session user is not in react-query: editing yourself on the users page must reach it too. */
function refreshSessionUser(edited: User) {
    const { user, checkAuth } = useAuthStore.getState()
    if (user?.id === edited.id) {
        void checkAuth()
    }
}

export function useUsers() {
    return useQuery({
        queryKey: QUERY_KEY,
        queryFn: () => usersApi.getAll(),
    })
}

export function useUser(id: string | number) {
    return useResourceItem(QUERY_KEY, () => usersApi.getById(id), id)
}

export function useCreateUser() {
    return useResourceMutation({
        mutationFn: (data: UserFormData) => usersApi.create(data),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.user.created'),
    })
}

export function useUpdateUser() {
    return useResourceMutation({
        mutationFn: ({ id, data }: { id: string | number; data: Partial<UserFormData> }) =>
            usersApi.update(id, data),
        invalidateKeys: [QUERY_KEY],
        onSuccess: refreshSessionUser,
        successMessage: i18n.t('toasts.user.updated'),
    })
}

export function useSetEmailVerified() {
    return useResourceMutation({
        mutationFn: ({ id, verified }: { id: number; verified: boolean }) => usersApi.setEmailVerified(id, verified),
        invalidateKeys: [QUERY_KEY],
        onSuccess: refreshSessionUser,
        successMessage: i18n.t('toasts.user.updated'),
    })
}

export function useDeleteUser() {
    return useResourceMutation({
        mutationFn: (id: string | number) => usersApi.delete(id),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.user.deleted'),
    })
}

export function useIssuePasswordToken() {
    return useResourceMutation({
        mutationFn: ({ id, send, locale }: { id: string | number; send?: boolean; locale?: string }) =>
            usersApi.issuePasswordToken(id, send, locale),
        invalidateKeys: [QUERY_KEY],
        successMessage: (user, { send }) =>
            send ? i18n.t('toasts.user.resetEmailSent', { email: user.email }) : i18n.t('toasts.user.passwordLink'),
    })
}
