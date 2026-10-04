import { api } from './client'
import { Space } from '@/types/spaces'

export const spacesApi = {
    list: () => api.get<Space[]>('/spaces'),
    create: (name: string) => api.post<Space, { name: string }>('/spaces', { name }),
    settings: (id: number) => api.get<Record<string, unknown>>(`/spaces/${id}/settings`),
    updateSettings: (id: number, data: Record<string, unknown>) =>
        api.patch<Record<string, unknown>, Record<string, unknown>>(`/spaces/${id}/settings`, data),
}
