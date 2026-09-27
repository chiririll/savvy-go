import { api, apiClient } from './client'
import { Backup, BackupInspection } from '@/types/backup'

const ENDPOINT = '/backups'

export const backupsApi = {
    getAll: () =>
        api.get<Backup[]>(ENDPOINT),

    create: (note?: string) =>
        api.post<Backup, { note?: string }>(ENDPOINT, { note }),

    upload: async (file: File, note?: string): Promise<Backup> => {
        const formData = new FormData()
        formData.append('file', file)
        if (note) formData.append('note', note)

        const response = await apiClient.post(`${ENDPOINT}/upload`, formData, {
            headers: { 'Content-Type': 'multipart/form-data' },
        })
        return response.data
    },

    download: (id: number) =>
        `${apiClient.defaults.baseURL}${ENDPOINT}/${id}/download`,

    inspect: (id: number) =>
        api.get<BackupInspection>(`${ENDPOINT}/${id}/inspect`),

    restore: (id: number) =>
        api.post<{ message: string }, void>(`${ENDPOINT}/${id}/restore`, undefined),

    sync: () =>
        api.post<{ removed_dead: number; registered_orphans: number }, void>(`${ENDPOINT}/sync`, undefined),

    delete: (id: number) =>
        api.delete<void>(`${ENDPOINT}/${id}`),
}
