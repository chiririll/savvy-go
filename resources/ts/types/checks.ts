export const CHECK_SEVERITIES = ['error', 'warning', 'info'] as const
export type CheckSeverity = (typeof CHECK_SEVERITIES)[number]

/** Ids of `GET /admin/checks`; the wording and the link to the fix live in the SPA. */
export const CHECK_IDS = [
    'app_url_missing',
    'mail_off',
    'mail_invalid',
    'backup_missing',
    'backup_stale',
    'disk_filling',
    'config_read_only',
] as const
export type CheckId = (typeof CHECK_IDS)[number]

export interface ServerCheck {
    id: CheckId
    severity: CheckSeverity
    /** Number shown in the message (disk percent, days since a backup), if any. */
    value: number | null
}
