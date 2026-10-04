import { intlLocale } from '@/lib/i18n'

export function formatBytes(bytes: number): string {
    if (!bytes) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), sizes.length - 1)
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

export function formatDateTime(value: string | null | undefined): string {
    return value ? new Date(value).toLocaleString(intlLocale()) : ''
}
