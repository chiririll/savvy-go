import { useTranslation } from 'react-i18next'

import type { ResourceSnapshot } from '@/types/monitoring'

import { Gauge } from './Gauge'
import { Caption } from './parts'
import { cn } from '@/lib/utils'

export function CpuCard({ cpu }: { cpu: ResourceSnapshot['cpu'] }) {
    const { t } = useTranslation('settings')
    const loadTitles = [t('monitoring.load1'), t('monitoring.load5'), t('monitoring.load15')]

    return (
        <div className="flex items-center gap-6 rounded-xl border bg-muted/20 p-5">
            <Gauge percent={cpu.load_percent} label={t('monitoring.cpuLoad')} size={150} />
            <div className="min-w-0 space-y-3 text-sm">
                <div>
                    <Caption>{t('monitoring.vcpus')}</Caption>
                    <p className="mt-0.5 text-lg font-semibold tabular-nums">{cpu.cores ?? '—'}</p>
                </div>
                <div>
                    <Caption>{t('monitoring.loadAvg')}</Caption>
                    <div className="mt-1 flex gap-1.5">
                        {cpu.load ? (
                            cpu.load.map((v, i) => (
                                <span
                                    key={i}
                                    className={cn(
                                        'rounded-md border bg-background px-2 py-0.5 font-mono text-xs tabular-nums',
                                    )}
                                    title={loadTitles[i]}
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
    )
}
