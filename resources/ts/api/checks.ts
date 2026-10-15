import { api } from './client'
import type { ServerCheck } from '@/types/checks'

export const checksApi = {
    list: () => api.get<ServerCheck[]>('/admin/checks'),
}
