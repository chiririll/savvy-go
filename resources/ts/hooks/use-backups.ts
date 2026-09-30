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
        mutationFn: (filename: string) => backupsApi.restore(filename),
        invalidateAll: true,
        successMessage: i18n.t('toasts.backup.restored'),
    })
}

export function useDeleteBackup() {
    return useResourceMutation({
        mutationFn: (filename: string) => backupsApi.delete(filename),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.backup.deleted'),
    })
}
