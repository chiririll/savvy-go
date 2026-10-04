import { api, apiClient } from './client'
import { Backup } from '@/types/backup'

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

    download: (filename: string) =>
        `${apiClient.defaults.baseURL}${ENDPOINT}/${encodeURIComponent(filename)}/download`,

    restore: (filename: string) =>
        api.post<{ message: string }, void>(`${ENDPOINT}/${encodeURIComponent(filename)}/restore`, undefined),

    delete: (filename: string) =>
        api.delete<void>(`${ENDPOINT}/${encodeURIComponent(filename)}`),
}
