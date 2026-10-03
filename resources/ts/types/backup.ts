export type BackupStatus =
    | 'current'
    | 'outdated'
    | 'newer'
    | 'legacy'
    | 'legacyUnsupported'
    | 'invalid'

export interface Backup {
    filename: string
    size: number
    note: string | null
    appVersion: string | null
    status: BackupStatus
    /** Migrations restore applies; non-zero only for 'outdated'. */
    pendingCount: number
    createdAt: string
}
