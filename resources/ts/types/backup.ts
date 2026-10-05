/**
 * current: signed by this server; unsigned: made elsewhere or edited;
 * raw: a bare database file (single-file layout or Laravel); invalid: unreadable.
 */
export const BACKUP_STATUSES = ['current', 'unsigned', 'raw', 'invalid'] as const
export type BackupStatus = (typeof BACKUP_STATUSES)[number]

export interface Backup {
    filename: string
    size: number
    note: string | null
    appVersion: string | null
    kind: 'server' | 'space' | ''
    spaceName?: string
    status: BackupStatus
    signature: 'own' | 'trusted' | 'unsigned'
    /** Whether this app can try to restore it; decided by the server. */
    restorable: boolean
    createdAt: string
}
