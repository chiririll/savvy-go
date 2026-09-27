import { useQuery } from '@tanstack/react-query'
import { backupsApi } from '@/api/backups'
import { useResourceMutation } from './use-crud'
import i18n from '@/lib/i18n'

const QUERY_KEY = ['backups']

export function useBackups() {
    return useQuery({
        queryKey: QUERY_KEY,
        queryFn: backupsApi.getAll,
    })
}

export function useCreateBackup() {
    return useResourceMutation({
        mutationFn: (note?: string) => backupsApi.create(note),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.backup.created'),
    })
}

export function useUploadBackup() {
    return useResourceMutation({
        mutationFn: ({ file, note }: { file: File; note?: string }) =>
            backupsApi.upload(file, note),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.backup.uploaded'),
    })
}

export function useRestoreBackup() {
    return useResourceMutation({
        mutationFn: (id: number) => backupsApi.restore(id),
        invalidateAll: true,
        successMessage: i18n.t('toasts.backup.restored'),
    })
}

export function useDeleteBackup() {
    return useResourceMutation({
        mutationFn: (id: number) => backupsApi.delete(id),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.backup.deleted'),
    })
}

export function useSyncBackups() {
    return useResourceMutation({
        mutationFn: () => backupsApi.sync(),
        invalidateKeys: [QUERY_KEY],
        successMessage: (data) => {
            if (data.removed_dead === 0 && data.registered_orphans === 0) {
                return i18n.t('toasts.backup.syncClean')
            }
            return i18n.t('toasts.backup.synced', {
                removed: data.removed_dead,
                found: data.registered_orphans,
            })
        },
    })
}
