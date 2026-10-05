import { useQuery, useQueryClient } from '@tanstack/react-query'
import i18n from '@/lib/i18n'
import { adminSpacesApi, spacesApi } from '@/api/spaces'
import { useUser } from '@/stores/auth'
import { useCurrentSpaceId, useSpaceStore } from '@/stores/space'
import { LinkedSpace, SpaceRole, SpaceTransferInput } from '@/types/spaces'
import { useResourceMutation } from './use-crud'

export const SPACES_KEY = ['spaces']
const spaceKey = (id: number | null, ...rest: unknown[]) => ['space', id, ...rest]

export function useSpaces() {
    const user = useUser()
    return useQuery({
        queryKey: [...SPACES_KEY, user?.id],
        queryFn: spacesApi.list,
        enabled: !!user,
    })
}

/** The space data requests go to. */
export function useCurrentSpace() {
    const id = useCurrentSpaceId()
    return useSpaces().data?.find((space) => space.id === id) ?? null
}

/** Whether the caller administers the current space. */
export function useIsSpaceAdmin() {
    return useCurrentSpace()?.role === 'admin'
}

/**
 * Switches the current space. Every cached query of the old space is reset,
 * so the page on screen fetches its data again from the new one. The space
 * list is refetched first: SpaceGate would otherwise see an id it does not
 * know yet (a space just created) and fall back to the first one.
 */
export function useSwitchSpace() {
    const queryClient = useQueryClient()
    const setCurrentId = useSpaceStore((state) => state.setCurrentId)
    return async (id: number) => {
        await queryClient.refetchQueries({ queryKey: SPACES_KEY })
        setCurrentId(id)
        await queryClient.resetQueries({ predicate: (q) => q.queryKey[0] !== SPACES_KEY[0] && q.queryKey[0] !== 'auth' })
    }
}

/**
 * Where to stay after switching space: the same page, without its query
 * string (filters and ?edit= point at the old space's records) and without
 * a record id in the path (/automation/12/logs becomes /automation).
 */
export function pathAfterSwitch(pathname: string): string {
    const parts = pathname.split('/')
    const id = parts.findIndex((part) => /^\d+$/.test(part))
    if (id > 0 && !pathname.startsWith('/admin/')) {
        return parts.slice(0, id).join('/') || '/'
    }
    return pathname
}

export function useCreateSpace() {
    return useResourceMutation({
        mutationFn: (name: string) => spacesApi.create(name),
        invalidateKeys: [SPACES_KEY],
        successMessage: i18n.t('spaces.toasts.created', { ns: 'settings' }),
    })
}

export function useImportSpace() {
    return useResourceMutation({
        mutationFn: ({ file, name }: { file: File; name?: string }) => spacesApi.importSpace(file, name),
        invalidateKeys: [SPACES_KEY],
        successMessage: i18n.t('spaces.toasts.imported', { ns: 'settings' }),
    })
}

export function useSpaceDetails(id: number | null) {
    return useQuery({ queryKey: spaceKey(id), queryFn: () => spacesApi.get(id!), enabled: id !== null })
}

export function useRenameSpace(id: number) {
    return useResourceMutation({
        mutationFn: (name: string) => spacesApi.rename(id, name),
        invalidateKeys: [SPACES_KEY, spaceKey(id)],
        successMessage: i18n.t('spaces.toasts.renamed', { ns: 'settings' }),
    })
}

export function useDeleteSpace(id: number) {
    return useResourceMutation({
        mutationFn: () => spacesApi.remove(id),
        invalidateKeys: [SPACES_KEY],
        successMessage: i18n.t('spaces.toasts.deleted', { ns: 'settings' }),
    })
}

export function useLeaveSpace(id: number) {
    return useResourceMutation({
        mutationFn: () => spacesApi.leave(id),
        invalidateKeys: [SPACES_KEY],
        successMessage: i18n.t('spaces.toasts.left', { ns: 'settings' }),
    })
}

export function useSpaceMembers(id: number | null) {
    return useQuery({ queryKey: spaceKey(id, 'members'), queryFn: () => spacesApi.members(id!), enabled: id !== null })
}

export function useSetMemberRole(id: number) {
    return useResourceMutation({
        mutationFn: ({ userId, role }: { userId: number; role: SpaceRole }) => spacesApi.setRole(id, userId, role),
        invalidateKeys: [spaceKey(id, 'members'), SPACES_KEY],
        successMessage: i18n.t('spaces.toasts.roleChanged', { ns: 'settings' }),
    })
}

export function useRemoveMember(id: number) {
    return useResourceMutation({
        mutationFn: (userId: number) => spacesApi.removeMember(id, userId),
        invalidateKeys: [spaceKey(id, 'members')],
        successMessage: i18n.t('spaces.toasts.memberRemoved', { ns: 'settings' }),
    })
}

export function useSpaceInvitations(id: number | null, enabled = true) {
    return useQuery({
        queryKey: spaceKey(id, 'invitations'),
        queryFn: () => spacesApi.invitations(id!),
        enabled: id !== null && enabled,
    })
}

export function useInvite(id: number) {
    return useResourceMutation({
        mutationFn: ({ role, email }: { role: SpaceRole; email?: string }) => spacesApi.invite(id, role, email),
        invalidateKeys: [spaceKey(id, 'invitations')],
    })
}

export function useRevokeInvitation(id: number) {
    return useResourceMutation({
        mutationFn: (invitationId: number) => spacesApi.revokeInvitation(id, invitationId),
        invalidateKeys: [spaceKey(id, 'invitations')],
        successMessage: i18n.t('spaces.toasts.invitationRevoked', { ns: 'settings' }),
    })
}

export function useSpaceLinks(id: number | null) {
    return useQuery({ queryKey: spaceKey(id, 'links'), queryFn: () => spacesApi.links(id!), enabled: id !== null })
}

/**
 * Linked spaces the current one can send money to: a transfer writes to
 * both, so the user must be able to write in each.
 */
export function useTransferTargets(): LinkedSpace[] {
    const current = useCurrentSpace()
    const { data: spaces } = useSpaces()
    const { data: links } = useSpaceLinks(current?.id ?? null)
    if (!current || current.role === 'viewer') {
        return []
    }
    return (links ?? []).filter((link) => {
        const role = spaces?.find((space) => space.id === link.id)?.role
        return role === 'admin' || role === 'editor'
    })
}

export function useLinkSpace(id: number) {
    return useResourceMutation({
        mutationFn: (targetId: number) => spacesApi.link(id, targetId),
        invalidateAll: true,
        successMessage: i18n.t('spaces.toasts.linked', { ns: 'settings' }),
    })
}

export function useUnlinkSpace(id: number) {
    return useResourceMutation({
        mutationFn: (otherId: number) => spacesApi.unlink(id, otherId),
        invalidateAll: true,
        successMessage: i18n.t('spaces.toasts.unlinked', { ns: 'settings' }),
    })
}

export function useSpaceTransfers(id: number | null) {
    return useQuery({ queryKey: spaceKey(id, 'transfers'), queryFn: () => spacesApi.transfers(id!), enabled: id !== null })
}

/** Sends money from the current space to a linked one. */
export function useSendTransfer() {
    const id = useCurrentSpaceId()
    return useResourceMutation({
        mutationFn: (data: SpaceTransferInput) => spacesApi.sendTransfer(id!, data),
        invalidateAll: true,
        successMessage: i18n.t('spaces.toasts.transferSent', { ns: 'settings' }),
    })
}

export function useResolveTransfer(id: number) {
    return useResourceMutation({
        mutationFn: ({ uuid, decision }: { uuid: string; decision: 'accept' | 'reject' | 'trust' }) =>
            spacesApi.resolveTransfer(id, uuid, decision),
        invalidateAll: true,
        successMessage: i18n.t('spaces.toasts.transferResolved', { ns: 'settings' }),
    })
}

export function useUpdateTransfer(id: number) {
    return useResourceMutation({
        mutationFn: ({ uuid, data }: { uuid: string; data: Partial<SpaceTransferInput> }) => spacesApi.updateTransfer(id, uuid, data),
        invalidateAll: true,
        successMessage: i18n.t('spaces.toasts.transferUpdated', { ns: 'settings' }),
    })
}

export function useDeleteTransfer(id: number) {
    return useResourceMutation({
        mutationFn: (uuid: string) => spacesApi.deleteTransfer(id, uuid),
        invalidateAll: true,
        successMessage: i18n.t('spaces.toasts.transferDeleted', { ns: 'settings' }),
    })
}

export function useSpaceAudit(id: number | null, enabled = true) {
    return useQuery({ queryKey: spaceKey(id, 'audit'), queryFn: () => spacesApi.audit(id!), enabled: id !== null && enabled })
}

export function useSpaceBackups(id: number | null) {
    return useQuery({ queryKey: spaceKey(id, 'backups'), queryFn: () => spacesApi.backups(id!), enabled: id !== null })
}

// --- server admin ----------------------------------------------------------

const ADMIN_SPACES = ['admin', 'spaces']

export function useAdminSpaces() {
    return useQuery({ queryKey: ADMIN_SPACES, queryFn: adminSpacesApi.list })
}

export function useDeletedSpaces() {
    return useQuery({ queryKey: [...ADMIN_SPACES, 'deleted'], queryFn: adminSpacesApi.deleted })
}

export function useSetSpaceQuota() {
    return useResourceMutation({
        mutationFn: ({ id, quotaBytes }: { id: number; quotaBytes: number | null }) => adminSpacesApi.setQuota(id, quotaBytes),
        invalidateKeys: [ADMIN_SPACES],
        successMessage: i18n.t('admin.spaces.toasts.quota', { ns: 'settings' }),
    })
}

export function useAdminDeleteSpace() {
    return useResourceMutation({
        mutationFn: (id: number) => adminSpacesApi.remove(id),
        invalidateKeys: [ADMIN_SPACES, SPACES_KEY],
        successMessage: i18n.t('admin.spaces.toasts.deleted', { ns: 'settings' }),
    })
}

export function useAssignSpaceAdmin() {
    return useResourceMutation({
        mutationFn: ({ id, userId }: { id: number; userId: number }) => adminSpacesApi.assignAdmin(id, userId),
        invalidateKeys: [ADMIN_SPACES, SPACES_KEY],
        successMessage: i18n.t('admin.spaces.toasts.assigned', { ns: 'settings' }),
    })
}

export function useRestoreDeletedSpace() {
    return useResourceMutation({
        mutationFn: (name: string) => adminSpacesApi.restoreDeleted(name),
        invalidateKeys: [ADMIN_SPACES, SPACES_KEY],
        successMessage: i18n.t('admin.spaces.toasts.restored', { ns: 'settings' }),
    })
}

const KEYS = ['admin', 'keys']

export function useSigningKeys() {
    return useQuery({ queryKey: KEYS, queryFn: adminSpacesApi.keys })
}

export function useTrustKey() {
    return useResourceMutation({
        mutationFn: ({ name, publicKey }: { name: string; publicKey: string }) => adminSpacesApi.trustKey(name, publicKey),
        invalidateKeys: [KEYS],
        successMessage: i18n.t('admin.keys.toasts.trusted', { ns: 'settings' }),
    })
}

export function useUntrustKey() {
    return useResourceMutation({
        mutationFn: (kid: string) => adminSpacesApi.untrustKey(kid),
        invalidateKeys: [KEYS],
        successMessage: i18n.t('admin.keys.toasts.untrusted', { ns: 'settings' }),
    })
}

export function useRotateKey() {
    return useResourceMutation({
        mutationFn: () => adminSpacesApi.rotateKey(),
        invalidateKeys: [KEYS],
        successMessage: i18n.t('admin.keys.toasts.rotated', { ns: 'settings' }),
    })
}
