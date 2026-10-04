import { api, apiClient } from './client'
import { toQueryString } from '@/lib/query-string'
import { Backup } from '@/types/backup'
import { Account } from '@/types'
import {
    AuditEntry,
    InvitationPreview,
    LinkedSpace,
    SigningKeys,
    Space,
    SpaceDetails,
    SpaceInvitation,
    SpaceMember,
    SpaceOverview,
    SpaceRole,
    SpaceTransfer,
    SpaceTransferInput,
} from '@/types/spaces'

const base = (id: number) => `/spaces/${id}`

export const spacesApi = {
    list: () => api.get<Space[]>('/spaces'),
    create: (name: string) => api.post<Space, { name: string }>('/spaces', { name }),
    get: (id: number) => api.get<SpaceDetails>(base(id)),
    rename: (id: number, name: string) => api.patch<SpaceDetails, { name: string }>(base(id), { name }),
    remove: (id: number) => api.delete<void>(base(id)),
    leave: (id: number) => api.post<void>(`${base(id)}/leave`),

    settings: (id: number) => api.get<Record<string, unknown>>(`${base(id)}/settings`),
    updateSettings: (id: number, data: Record<string, unknown>) =>
        api.patch<Record<string, unknown>, Record<string, unknown>>(`${base(id)}/settings`, data),

    members: (id: number) => api.get<SpaceMember[]>(`${base(id)}/members`),
    setRole: (id: number, userId: number, role: SpaceRole) =>
        api.patch<SpaceMember[], { role: SpaceRole }>(`${base(id)}/members/${userId}`, { role }),
    removeMember: (id: number, userId: number) => api.delete<void>(`${base(id)}/members/${userId}`),

    invitations: (id: number) => api.get<SpaceInvitation[]>(`${base(id)}/invitations`),
    invite: (id: number, role: SpaceRole, email?: string) =>
        api.post<{ token: string }, { role: SpaceRole; email?: string }>(`${base(id)}/invitations`, { role, email }),
    revokeInvitation: (id: number, invitationId: number) => api.delete<void>(`${base(id)}/invitations/${invitationId}`),

    links: (id: number) => api.get<LinkedSpace[]>(`${base(id)}/links`),
    link: (id: number, targetSpaceId: number) =>
        api.post<LinkedSpace[], { target_space_id: number }>(`${base(id)}/links`, { target_space_id: targetSpaceId }),
    unlink: (id: number, otherId: number) => api.delete<void>(`${base(id)}/links/${otherId}`),

    /** Accounts of any space the caller belongs to (for the other side of a transfer). */
    accounts: (id: number, params?: { active?: boolean; exclude_debts?: boolean }) =>
        api.get<Account[]>(`${base(id)}/accounts${toQueryString(params)}`),

    transfers: (id: number) => api.get<SpaceTransfer[]>(`${base(id)}/transfers`),
    sendTransfer: (id: number, data: SpaceTransferInput) =>
        api.post<SpaceTransfer, SpaceTransferInput>(`${base(id)}/transfers`, data),
    updateTransfer: (id: number, uuid: string, data: Partial<SpaceTransferInput>) =>
        api.patch<SpaceTransfer, Partial<SpaceTransferInput>>(`${base(id)}/transfers/${uuid}`, data),
    deleteTransfer: (id: number, uuid: string) => api.delete<void>(`${base(id)}/transfers/${uuid}`),
    resolveTransfer: (id: number, uuid: string, decision: 'accept' | 'reject' | 'trust') =>
        api.post<SpaceTransfer | undefined>(`${base(id)}/transfers/${uuid}/${decision}`),

    audit: (id: number) => api.get<AuditEntry[]>(`${base(id)}/audit`),

    backups: (id: number) => api.get<Backup[]>(`${base(id)}/backups`),
    createBackup: (id: number, note?: string) => api.post<Backup, { note?: string }>(`${base(id)}/backups`, { note }),
    uploadBackup: async (id: number, file: File): Promise<Backup> => {
        const form = new FormData()
        form.append('file', file)
        const res = await apiClient.post(`${base(id)}/backups/upload`, form, {
            headers: { 'Content-Type': 'multipart/form-data' },
            timeout: 0,
        })
        return res.data?.data ?? res.data
    },
    backupDownloadUrl: (id: number, name: string) =>
        `${apiClient.defaults.baseURL}${base(id)}/backups/${encodeURIComponent(name)}/download`,
    restoreBackup: (id: number, name: string) =>
        api.post<{ message: string }>(`${base(id)}/backups/${encodeURIComponent(name)}/restore`, undefined, { timeout: 0 }),
    deleteBackup: (id: number, name: string) => api.delete<void>(`${base(id)}/backups/${encodeURIComponent(name)}`),

    /** A space backup (or an older database file) becomes a new space owned by the caller. */
    importSpace: async (file: File, name?: string): Promise<Space> => {
        const form = new FormData()
        form.append('file', file)
        if (name) form.append('name', name)
        const res = await apiClient.post('/spaces/import', form, {
            headers: { 'Content-Type': 'multipart/form-data' },
            timeout: 0,
        })
        return res.data?.data ?? res.data
    },
}

export const invitationsApi = {
    preview: (token: string) => api.get<InvitationPreview>(`/invitations/${token}`),
    accept: (token: string) => api.post<Space>(`/invitations/${token}/accept`),
    register: (token: string, data: { name: string; email: string; password: string }) =>
        api.post<unknown, typeof data>(`/invitations/${token}/register`, data),
}

export const adminSpacesApi = {
    list: () => api.get<SpaceOverview[]>('/admin/spaces'),
    setQuota: (id: number, quotaBytes: number | null) =>
        api.patch<SpaceOverview[], { quota_bytes: number | null }>(`/admin/spaces/${id}`, { quota_bytes: quotaBytes }),
    remove: (id: number) => api.delete<void>(`/admin/spaces/${id}`),
    assignAdmin: (id: number, userId: number) =>
        api.post<void, { user_id: number }>(`/admin/spaces/${id}/admins`, { user_id: userId }),
    deleted: () => api.get<Backup[]>('/admin/deleted-spaces'),
    restoreDeleted: (name: string) => api.post<Space>(`/admin/deleted-spaces/${encodeURIComponent(name)}/restore`),
    keys: () => api.get<SigningKeys>('/admin/keys'),
    trustKey: (name: string, publicKey: string) =>
        api.post<SigningKeys, { name: string; public_key: string }>('/admin/keys', { name, public_key: publicKey }),
    untrustKey: (kid: string) => api.delete<void>(`/admin/keys/${kid}`),
    rotateKey: () => api.post<SigningKeys>('/admin/keys/rotate'),
}
