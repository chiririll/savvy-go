import { Clock, Cpu, Layers, MemoryStick, Recycle } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { ResourceSnapshot } from '@/types/monitoring'

import { CpuCard } from './CpuCard'
import { formatBytes, formatCount, formatUptime } from './format'
import { MemoryCard } from './MemoryCard'
import { PanelHeader, PanelSubtitle, StatCard } from './parts'
import { severityFor } from './severity'
import { cn } from '@/lib/utils'

function RuntimeChip({ label, value }: { label: string; value: string }) {
    return (
        <span
            className={cn(
                'inline-flex items-center gap-1.5 rounded-full border bg-muted/40 px-2.5 py-1',
                'text-[11px] font-medium',
            )}
        >
            <span className="text-muted-foreground">{label}</span>
            <span className="font-mono">{value}</span>
        </span>
    )
}

export function ResourcePanel({ data }: { data: ResourceSnapshot }) {
    const { t } = useTranslation('settings')
    const { cpu, memory, process, runtime } = data
    const cpuSeverity = severityFor(cpu.load_percent)
    const processMemoryHint =
        process.limit_bytes && process.limit_bytes > 0
            ? t('monitoring.ofLimit', { percent: ((process.memory_bytes / process.limit_bytes) * 100).toFixed(0) })
            : t('monitoring.peak', { size: formatBytes(process.peak_bytes) })

    return (
        <section className="rounded-2xl border bg-card shadow-sm">
            <PanelHeader
                icon={Cpu}
                title={t('monitoring.resourcesTitle')}
                subtitle={<PanelSubtitle>{t('monitoring.resourcesDescription')}</PanelSubtitle>}
            >
                <div className="ml-auto flex flex-wrap items-center gap-2">
                    <RuntimeChip label="Go" value={runtime.go_version.replace(/^go/, '')} />
                    <RuntimeChip label="Savvy" value={runtime.app_version} />
                    <RuntimeChip label="env" value={runtime.environment} />
                </div>
            </PanelHeader>

            <div className="grid grid-cols-1 gap-6 p-6 md:grid-cols-2">
                <CpuCard cpu={cpu} />
                <MemoryCard memory={memory} />
            </div>

            <div className="grid grid-cols-1 gap-4 border-t p-6 sm:grid-cols-2 lg:grid-cols-4">
                <StatCard
                    icon={MemoryStick}
                    label={t('monitoring.processMemory')}
                    value={formatBytes(process.memory_bytes)}
                    hint={processMemoryHint}
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
