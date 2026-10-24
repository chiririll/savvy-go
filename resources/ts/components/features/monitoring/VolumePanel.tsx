import { AlertTriangle, HardDrive } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'
import type { StorageSnapshot } from '@/types/monitoring'

import { formatBytes } from './format'
import { Gauge } from './Gauge'
import { Caption, LegendDot, PanelHeader } from './parts'
import { SEVERITY_FILL, severityFor, type Severity } from './severity'

function SeverityBadge({ severity }: { severity: Exclude<Severity, 'ok'> }) {
    const { t } = useTranslation('settings')
    return (
        <span
            className={cn(
                'ml-auto inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px]',
                'font-semibold uppercase tracking-wide',
                severity === 'crit'
                    ? 'bg-destructive/10 text-destructive'
                    : 'bg-warning/10 text-warning',
            )}
        >
            <AlertTriangle className="size-3" />
            {severity === 'crit' ? t('monitoring.critical') : t('monitoring.fillingUp')}
        </span>
    )
}

function Total({ label, bytes }: { label: string; bytes: number }) {
    return (
        <div>
            <Caption>{label}</Caption>
            <p className="mt-1 text-xl font-semibold tabular-nums">{formatBytes(bytes)}</p>
        </div>
    )
}

interface VolumeDetailsProps {
    severity: Severity
    total: number
    used: number
    free: number
    appBytes: number
}

function VolumeDetails({ severity, total, used, free, appBytes }: VolumeDetailsProps) {
    const { t } = useTranslation('settings')
    const otherBytes = Math.max(0, used - appBytes)
    const segments =
        total > 0
            ? [
                  { className: SEVERITY_FILL[severity], width: (appBytes / total) * 100 },
                  { className: 'bg-muted-foreground/35', width: (otherBytes / total) * 100 },
              ]
            : []

    return (
        <div className="space-y-5">
            <div className="flex h-3 w-full overflow-hidden rounded-full bg-muted">
                {segments.map((segment, i) => (
                    <div
                        key={i}
                        className={cn('h-full transition-all duration-700', segment.className)}
                        style={{ width: `${segment.width}%` }}
                    />
                ))}
            </div>

            <div className="grid gap-2.5 text-sm sm:grid-cols-2">
                <LegendDot
                    className={SEVERITY_FILL[severity]}
                    label={t('monitoring.savvyData')}
                    value={formatBytes(appBytes)}
                />
                <LegendDot
                    className="bg-muted-foreground/35"
                    label={t('monitoring.otherUsage')}
                    value={formatBytes(otherBytes)}
                />
                <LegendDot className="bg-muted" label={t('monitoring.free')} value={formatBytes(free)} />
                <LegendDot className="bg-foreground" label={t('monitoring.capacity')} value={formatBytes(total)} />
            </div>

            <div className="grid grid-cols-3 gap-4 border-t pt-5">
                <Total label={t('monitoring.used')} bytes={used} />
                <Total label={t('monitoring.free')} bytes={free} />
                <Total label={t('monitoring.total')} bytes={total} />
            </div>
        </div>
    )
}

export function VolumePanel({ data }: { data: StorageSnapshot }) {
    const { t } = useTranslation('settings')
    const { volume, managed } = data
    const severity = severityFor(volume.used_percent)
    const used = volume.used_bytes ?? 0

    return (
        <section className="rounded-2xl border bg-card shadow-sm">
            <PanelHeader
                icon={HardDrive}
                align="start"
                title={t('monitoring.uploadVolume')}
                subtitle={
                    <p className="mt-1 truncate font-mono text-xs text-muted-foreground" title={volume.path}>
                        {volume.disk} · {volume.path}
                    </p>
                }
            >
                {severity !== 'ok' && <SeverityBadge severity={severity} />}
            </PanelHeader>

            {volume.total_bytes === null ? (
                <div className="flex items-center gap-3 px-6 py-10 text-sm text-muted-foreground">
                    <AlertTriangle className="size-4" />
                    {t('monitoring.volumeUnavailable')}
                </div>
            ) : (
                <div className="grid items-center gap-8 p-6 md:grid-cols-[auto_1fr]">
                    <div className="flex justify-center md:justify-start">
                        <Gauge percent={volume.used_percent} label={t('monitoring.used')} />
                    </div>
                    <VolumeDetails
                        severity={severity}
                        total={volume.total_bytes}
                        used={used}
                        free={volume.free_bytes ?? 0}
                        appBytes={Math.min(managed.used_bytes, used)}
                    />
                </div>
            )}
        </section>
    )
}
