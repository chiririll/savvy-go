import { useMemo } from 'react'
import {
    HardDrive,
    Database,
    Boxes,
    UploadCloud,
    FileSpreadsheet,
    RefreshCw,
    AlertTriangle,
    Cpu,
    MemoryStick,
    Layers,
    Clock,
    Recycle,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Page, PageHeader } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { useStorageMonitoring, useResourceMonitoring } from '@/hooks'
import type { StorageSnapshot, ResourceSnapshot } from '@/types/monitoring'
import i18n, { intlLocale } from '@/lib/i18n'

function formatBytes(bytes: number | null | undefined, decimals = 1): string {
    if (bytes === null || bytes === undefined) return '—'
    if (bytes === 0) return '0 B'
    const k = 1024
    const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
    const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), units.length - 1)
    const value = bytes / Math.pow(k, i)
    return `${parseFloat(value.toFixed(i === 0 ? 0 : decimals))} ${units[i]}`
}

function formatCount(value: number | null | undefined): string {
    if (value === null || value === undefined) return '—'
    return new Intl.NumberFormat(intlLocale()).format(value)
}

function formatUptime(seconds: number | null): string {
    if (seconds === null) return '—'
    const d = Math.floor(seconds / 86400)
    const h = Math.floor((seconds % 86400) / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    if (d > 0) return i18n.t('monitoring.uptimeDaysHours', { ns: 'settings', days: d, hours: h })
    if (h > 0) return i18n.t('monitoring.uptimeHoursMinutes', { ns: 'settings', hours: h, minutes: m })
    return i18n.t('monitoring.uptimeMinutes', { ns: 'settings', minutes: m })
}

type Severity = 'ok' | 'warn' | 'crit'

function severityFor(percent: number | null): Severity {
    if (percent === null) return 'ok'
    if (percent >= 90) return 'crit'
    if (percent >= 75) return 'warn'
    return 'ok'
}

const SEVERITY_STROKE: Record<Severity, string> = {
    ok: 'text-emerald-500',
    warn: 'text-amber-500',
    crit: 'text-red-500',
}

const SEVERITY_FILL: Record<Severity, string> = {
    ok: 'bg-emerald-500',
    warn: 'bg-amber-500',
    crit: 'bg-red-500',
}

function Gauge({
    percent,
    label = 'Used',
    size = 196,
}: {
    percent: number | null
    label?: string
    size?: number
}) {
    const severity = severityFor(percent)
    const stroke = size >= 180 ? 14 : 11
    const radius = (size - stroke) / 2
    const circumference = 2 * Math.PI * radius
    const clamped = Math.max(0, Math.min(100, percent ?? 0))
    const offset = circumference - (clamped / 100) * circumference

    return (
        <div className="relative" style={{ width: size, height: size }}>
            <svg width={size} height={size} className="-rotate-90">
                <circle
                    cx={size / 2}
                    cy={size / 2}
                    r={radius}
                    fill="none"
                    strokeWidth={stroke}
                    className="stroke-muted"
                />
                <circle
                    cx={size / 2}
                    cy={size / 2}
                    r={radius}
                    fill="none"
                    strokeWidth={stroke}
                    strokeLinecap="round"
                    strokeDasharray={circumference}
                    strokeDashoffset={offset}
                    className={cn('transition-[stroke-dashoffset] duration-700 ease-out', SEVERITY_STROKE[severity])}
                />
            </svg>
            <div className="absolute inset-0 flex flex-col items-center justify-center">
                <span className={cn('font-bold tabular-nums tracking-tight', size >= 180 ? 'text-4xl' : 'text-3xl')}>
                    {percent === null ? '—' : `${percent}%`}
                </span>
                <span className="mt-1 text-xs font-medium uppercase tracking-wider text-muted-foreground">
                    {label}
                </span>
            </div>
        </div>
    )
}

function StatCard({
    icon: Icon,
    label,
    value,
    hint,
    tone,
}: {
    icon: LucideIcon
    label: string
    value: string
    hint?: string
    tone?: Severity
}) {
    return (
        <div className="rounded-xl border bg-card p-4 shadow-sm">
            <div className="flex items-center gap-2 text-muted-foreground">
                <Icon className={cn('size-4', tone && tone !== 'ok' && SEVERITY_STROKE[tone])} />
                <span className="text-xs font-medium uppercase tracking-wide">{label}</span>
            </div>
            <p
                className={cn(
                    'mt-3 text-2xl font-semibold tabular-nums tracking-tight',
                    tone && tone !== 'ok' && SEVERITY_STROKE[tone]
                )}
            >
                {value}
            </p>
            {hint && <p className="mt-1 text-xs text-muted-foreground">{hint}</p>}
        </div>
    )
}

function LegendDot({ className, label, value }: { className: string; label: string; value: string }) {
    return (
        <div className="flex items-center gap-2">
            <span className={cn('size-2.5 shrink-0 rounded-sm', className)} />
            <span className="text-muted-foreground">{label}</span>
            <span className="ml-auto font-medium tabular-nums">{value}</span>
        </div>
    )
}

function RuntimeChip({ label, value }: { label: string; value: string }) {
    return (
        <span className="inline-flex items-center gap-1.5 rounded-full border bg-muted/40 px-2.5 py-1 text-[11px] font-medium">
            <span className="text-muted-foreground">{label}</span>
            <span className="font-mono">{value}</span>
        </span>
    )
}

function ResourcePanel({ data }: { data: ResourceSnapshot }) {
    const { t } = useTranslation('settings')
    const { cpu, memory, process, runtime } = data
    const memSeverity = severityFor(memory.used_percent)
    const cpuSeverity = severityFor(cpu.load_percent)
    const procPercent =
        process.limit_bytes && process.limit_bytes > 0
            ? t('monitoring.ofLimit', { percent: ((process.memory_bytes / process.limit_bytes) * 100).toFixed(0) })
            : t('monitoring.peak', { size: formatBytes(process.peak_bytes) })

    return (
        <section className="rounded-2xl border bg-card shadow-sm">
            <header className="flex flex-wrap items-center gap-3 border-b px-6 py-5">
                <div className="flex size-10 shrink-0 items-center justify-center rounded-xl border bg-muted/50 text-muted-foreground">
                    <Cpu className="size-5" />
                </div>
                <div>
                    <h2 className="text-sm font-semibold leading-none">{t('monitoring.resourcesTitle')}</h2>
                    <p className="mt-1 text-xs text-muted-foreground">{t('monitoring.resourcesDescription')}</p>
                </div>
                <div className="ml-auto flex flex-wrap items-center gap-2">
                    <RuntimeChip label="Go" value={runtime.go_version.replace(/^go/, '')} />
                    <RuntimeChip label="Savvy" value={runtime.app_version} />
                    <RuntimeChip label="env" value={runtime.environment} />
                </div>
            </header>

            <div className="grid gap-6 p-6 md:grid-cols-2">
                <div className="flex items-center gap-6 rounded-xl border bg-muted/20 p-5">
                    <Gauge percent={cpu.load_percent} label={t('monitoring.cpuLoad')} size={150} />
                    <div className="min-w-0 space-y-3 text-sm">
                        <div>
                            <p className="text-xs uppercase tracking-wide text-muted-foreground">{t('monitoring.vcpus')}</p>
                            <p className="mt-0.5 text-lg font-semibold tabular-nums">
                                {cpu.cores ?? '—'}
                            </p>
                        </div>
                        <div>
                            <p className="text-xs uppercase tracking-wide text-muted-foreground">{t('monitoring.loadAvg')}</p>
                            <div className="mt-1 flex gap-1.5">
                                {cpu.load ? (
                                    cpu.load.map((v, i) => (
                                        <span
                                            key={i}
                                            className="rounded-md border bg-background px-2 py-0.5 font-mono text-xs tabular-nums"
                                            title={[t('monitoring.load1'), t('monitoring.load5'), t('monitoring.load15')][i]}
                                        >
                                            {v.toFixed(2)}
                                        </span>
                                    ))
                                ) : (
                                    <span className="text-muted-foreground">{t('monitoring.unavailable')}</span>
                                )}
                            </div>
                        </div>
                    </div>
                </div>

                <div className="flex items-center gap-6 rounded-xl border bg-muted/20 p-5">
                    <Gauge percent={memory.used_percent} label={t('monitoring.memory')} size={150} />
                    <div className="min-w-0 space-y-3 text-sm">
                        {memory.total_bytes === null ? (
                            <p className="text-muted-foreground">{t('monitoring.memoryUnavailable')}</p>
                        ) : (
                            <>
                                <div className="space-y-2">
                                    <LegendDot
                                        className={SEVERITY_FILL[memSeverity]}
                                        label={t('monitoring.used')}
                                        value={formatBytes(memory.used_bytes)}
                                    />
                                    <LegendDot
                                        className="bg-muted"
                                        label={t('monitoring.free')}
                                        value={formatBytes(memory.free_bytes)}
                                    />
                                    <LegendDot
                                        className="bg-foreground"
                                        label={t('monitoring.limit')}
                                        value={formatBytes(memory.total_bytes)}
                                    />
                                </div>
                                {memory.source && (
                                    <p className="text-xs text-muted-foreground">
                                        {memory.source === 'cgroup' ? t('monitoring.cgroupLimit') : t('monitoring.hostMemory')}
                                    </p>
                                )}
                            </>
                        )}
                    </div>
                </div>
            </div>

            <div className="grid gap-4 border-t p-6 sm:grid-cols-2 lg:grid-cols-4">
                <StatCard
                    icon={MemoryStick}
                    label={t('monitoring.processMemory')}
                    value={formatBytes(process.memory_bytes)}
                    hint={procPercent}
                />
                <StatCard
                    icon={Layers}
                    label={t('monitoring.goroutines')}
                    value={formatCount(process.goroutines)}
                    hint={t('monitoring.goroutinesHint')}
                />
                <StatCard
                    icon={Recycle}
                    label={t('monitoring.gcCycles')}
                    value={formatCount(process.gc_cycles)}
                    hint={t('monitoring.gcCyclesHint')}
                />
                <StatCard
                    icon={Clock}
                    label={t('monitoring.uptime')}
                    value={formatUptime(runtime.uptime_seconds)}
                    hint={cpuSeverity === 'crit' ? t('monitoring.highCpu') : undefined}
                />
            </div>
        </section>
    )
}

function VolumePanel({ data }: { data: StorageSnapshot }) {
    const { t } = useTranslation('settings')
    const { volume, managed } = data
    const severity = severityFor(volume.used_percent)
    const total = volume.total_bytes ?? 0
    const used = volume.used_bytes ?? 0
    const free = volume.free_bytes ?? 0
    const appBytes = Math.min(managed.used_bytes, used)
    const otherBytes = Math.max(0, used - appBytes)

    const segments = total > 0
        ? [
              { className: SEVERITY_FILL[severity], width: (appBytes / total) * 100 },
              { className: 'bg-muted-foreground/35', width: (otherBytes / total) * 100 },
          ]
        : []

    return (
        <section className="rounded-2xl border bg-card shadow-sm">
            <header className="flex items-start gap-3 border-b px-6 py-5">
                <div className="flex size-10 shrink-0 items-center justify-center rounded-xl border bg-muted/50 text-muted-foreground">
                    <HardDrive className="size-5" />
                </div>
                <div className="min-w-0">
                    <h2 className="text-sm font-semibold leading-none">{t('monitoring.uploadVolume')}</h2>
                    <p className="mt-1 truncate font-mono text-xs text-muted-foreground" title={volume.path}>
                        {volume.disk} · {volume.path}
                    </p>
                </div>
                {severity !== 'ok' && (
                    <span
                        className={cn(
                            'ml-auto inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-semibold uppercase tracking-wide',
                            severity === 'crit'
                                ? 'bg-red-500/10 text-red-600 dark:text-red-400'
                                : 'bg-amber-500/10 text-amber-600 dark:text-amber-400'
                        )}
                    >
                        <AlertTriangle className="size-3" />
                        {severity === 'crit' ? t('monitoring.critical') : t('monitoring.fillingUp')}
                    </span>
                )}
            </header>

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
                            <div>
                                <p className="text-xs uppercase tracking-wide text-muted-foreground">{t('monitoring.used')}</p>
                                <p className="mt-1 text-xl font-semibold tabular-nums">{formatBytes(used)}</p>
                            </div>
                            <div>
                                <p className="text-xs uppercase tracking-wide text-muted-foreground">{t('monitoring.free')}</p>
                                <p className="mt-1 text-xl font-semibold tabular-nums">{formatBytes(free)}</p>
                            </div>
                            <div>
                                <p className="text-xs uppercase tracking-wide text-muted-foreground">{t('monitoring.total')}</p>
                                <p className="mt-1 text-xl font-semibold tabular-nums">{formatBytes(total)}</p>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </section>
    )
}

function BucketBreakdown({ data }: { data: StorageSnapshot }) {
    const { t } = useTranslation('settings')
    const { buckets, used_bytes } = data.managed

    return (
        <section className="rounded-2xl border bg-card shadow-sm">
            <header className="flex items-center gap-3 border-b px-6 py-5">
                <div className="flex size-10 shrink-0 items-center justify-center rounded-xl border bg-muted/50 text-muted-foreground">
                    <Boxes className="size-5" />
                </div>
                <div>
                    <h2 className="text-sm font-semibold leading-none">{t('monitoring.storageByBucket')}</h2>
                    <p className="mt-1 text-xs text-muted-foreground">{t('monitoring.storageByBucketDescription')}</p>
                </div>
            </header>

            {buckets.length === 0 ? (
                <p className="px-6 py-10 text-center text-sm text-muted-foreground">
                    {t('monitoring.nothingStored')}
                </p>
            ) : (
                <ul className="divide-y">
                    {buckets.map((bucket) => {
                        const share = used_bytes > 0 ? (bucket.bytes / used_bytes) * 100 : 0
                        return (
                            <li key={bucket.bucket} className="px-6 py-4">
                                <div className="flex items-baseline justify-between gap-4">
                                    <span className="truncate font-mono text-sm">{bucket.bucket}</span>
                                    <span className="shrink-0 text-sm font-semibold tabular-nums">
                                        {formatBytes(bucket.bytes)}
                                    </span>
                                </div>
                                <div className="mt-2 flex h-1.5 w-full overflow-hidden rounded-full bg-muted">
                                    <div
                                        className="h-full rounded-full bg-primary transition-all duration-700"
                                        style={{ width: `${share}%` }}
                                    />
                                </div>
                                <div className="mt-1.5 flex justify-between text-xs text-muted-foreground">
                                    <span>{t('monitoring.objects', { count: bucket.objects })}</span>
                                    <span className="tabular-nums">{share.toFixed(0)}%</span>
                                </div>
                            </li>
                        )
                    })}
                </ul>
            )}
        </section>
    )
}

function MonitoringSkeleton() {
    return (
        <div className="space-y-6">
            <Skeleton className="h-80 w-full rounded-2xl" />
            <Skeleton className="h-72 w-full rounded-2xl" />
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                {Array.from({ length: 4 }).map((_, i) => (
                    <Skeleton key={i} className="h-28 rounded-xl" />
                ))}
            </div>
        </div>
    )
}

export default function MonitoringPage() {
    const { t } = useTranslation('settings')
    const storage = useStorageMonitoring()
    const resources = useResourceMonitoring()

    const isInitialLoading = (storage.isLoading && !storage.data) || (resources.isLoading && !resources.data)
    const isFetching = storage.isFetching || resources.isFetching

    const updatedLabel = useMemo(() => {
        const latest = Math.max(storage.dataUpdatedAt ?? 0, resources.dataUpdatedAt ?? 0)
        return latest > 0 ? new Date(latest).toLocaleTimeString(intlLocale()) : null
    }, [storage.dataUpdatedAt, resources.dataUpdatedAt])

    const refresh = () => {
        storage.refetch()
        resources.refetch()
    }

    return (
        <Page title={t('monitoring.title')}>
            <PageHeader
                title={t('monitoring.title')}
                description={t('monitoring.description')}
                actions={
                    <div className="flex items-center gap-3">
                        {updatedLabel && (
                            <span className="hidden items-center gap-1.5 text-xs text-muted-foreground sm:flex">
                                <span className="relative flex size-2">
                                    <span className="absolute inline-flex size-full animate-ping rounded-full bg-emerald-500 opacity-60" />
                                    <span className="relative inline-flex size-2 rounded-full bg-emerald-500" />
                                </span>
                                {t('monitoring.updated', { time: updatedLabel })}
                            </span>
                        )}
                        <Button variant="outline" size="sm" onClick={refresh} disabled={isFetching}>
                            <RefreshCw className={cn('mr-2 size-4', isFetching && 'animate-spin')} />
                            {t('monitoring.refresh')}
                        </Button>
                    </div>
                }
            />

            {isInitialLoading ? (
                <MonitoringSkeleton />
            ) : (
                <div className="space-y-6">
                    {resources.data && <ResourcePanel data={resources.data} />}

                    {storage.data && (
                        <>
                            <VolumePanel data={storage.data} />

                            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                                <StatCard
                                    icon={Database}
                                    label={t('monitoring.savvyData')}
                                    value={formatBytes(storage.data.managed.used_bytes)}
                                    hint={t('monitoring.storedObjects', { count: storage.data.managed.objects })}
                                />
                                <StatCard
                                    icon={UploadCloud}
                                    label={t('monitoring.inFlight')}
                                    value={formatBytes(storage.data.managed.pending_bytes)}
                                    hint={t('monitoring.pending', { count: storage.data.uploads.by_status.pending ?? 0 })}
                                />
                                <StatCard
                                    icon={Boxes}
                                    label={t('monitoring.uploads')}
                                    value={formatCount(storage.data.uploads.total)}
                                    hint={t('monitoring.completed', { count: storage.data.uploads.by_status.completed ?? 0 })}
                                />
                                <StatCard
                                    icon={FileSpreadsheet}
                                    label={t('monitoring.imports')}
                                    value={formatCount(storage.data.imports.total)}
                                    hint={t('monitoring.completed', { count: storage.data.imports.by_status.completed ?? 0 })}
                                />
                            </div>

                            <BucketBreakdown data={storage.data} />
                        </>
                    )}
                </div>
            )}
        </Page>
    )
}
