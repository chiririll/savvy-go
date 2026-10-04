import { api } from './client'
import { Space } from '@/types/spaces'

export const spacesApi = {
    list: () => api.get<Space[]>('/spaces'),
}
