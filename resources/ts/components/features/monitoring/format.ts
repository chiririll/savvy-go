import i18n, { intlLocale } from '@/lib/i18n'

export function formatBytes(bytes: number | null | undefined, decimals = 1): string {
    if (bytes === null || bytes === undefined) return '—'
    if (bytes === 0) return '0 B'
    const k = 1024
    const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
    const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), units.length - 1)
    const value = bytes / Math.pow(k, i)
    return `${parseFloat(value.toFixed(i === 0 ? 0 : decimals))} ${units[i]}`
}

export function formatCount(value: number | null | undefined): string {
    if (value === null || value === undefined) return '—'
    return new Intl.NumberFormat(intlLocale()).format(value)
}

export function formatUptime(seconds: number | null): string {
    if (seconds === null) return '—'
    const d = Math.floor(seconds / 86400)
    const h = Math.floor((seconds % 86400) / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    if (d > 0) return i18n.t('monitoring.uptimeDaysHours', { ns: 'settings', days: d, hours: h })
    if (h > 0) return i18n.t('monitoring.uptimeHoursMinutes', { ns: 'settings', hours: h, minutes: m })
    return i18n.t('monitoring.uptimeMinutes', { ns: 'settings', minutes: m })
}
