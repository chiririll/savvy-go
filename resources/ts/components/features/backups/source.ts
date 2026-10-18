import type { Backup } from '@/types/backup'
import type { LaravelTarget } from './LaravelRestoreOptions'

/** Where backups come from: the server's, or one space's. */
export interface BackupSource {
    key: unknown[]
    list: () => Promise<Backup[]>
    create: (note?: string) => Promise<Backup>
    upload: (file: File, note?: string) => Promise<Backup>
    downloadUrl: (filename: string) => string
    restore: (filename: string) => Promise<unknown>
    remove: (filename: string) => Promise<unknown>
    /**
     * A Laravel database is not kept as it is: for the server or this space only the
     * upgraded version is saved as a backup, to be restored like any other. A new
     * space is created at once. The upload dialog asks which.
     */
    laravel?: {
        targets: LaravelTarget[]
        restore: (file: File, target: LaravelTarget, appKey?: string) => Promise<unknown>
    }
}
