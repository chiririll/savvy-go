export type BackupSchemaStatus = 'current' | 'outdated' | 'newer' | 'unknown'

export interface Backup {
    filename: string
    size: number
    note: string | null
    schemaVersion: string | null
    schemaStatus: BackupSchemaStatus
    createdAt: string
}

export interface BackupInspection {
    valid: boolean
    compatible: boolean
    incompatibleReason?: 'newer' | 'legacy_unsupported'
    legacy: boolean
    pendingCount: number
    pendingMigrations: string[]
    unknownCount: number
    unknownMigrations: string[]
}
