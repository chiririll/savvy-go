import { useTranslation } from 'react-i18next'

import type { ResourceSnapshot } from '@/types/monitoring'

import { formatBytes } from './format'
import { Gauge } from './Gauge'
import { LegendDot } from './parts'
import { SEVERITY_FILL, severityFor } from './severity'

export function MemoryCard({ memory }: { memory: ResourceSnapshot['memory'] }) {
    const { t } = useTranslation('settings')

    return (
        <div className="flex items-center gap-6 rounded-xl border bg-muted/20 p-5">
            <Gauge percent={memory.used_percent} label={t('monitoring.memory')} size={150} />
            <div className="min-w-0 space-y-3 text-sm">
                {memory.total_bytes === null ? (
                    <p className="text-muted-foreground">{t('monitoring.memoryUnavailable')}</p>
                ) : (
                    <>
                        <div className="space-y-2">
                            <LegendDot
                                className={SEVERITY_FILL[severityFor(memory.used_percent)]}
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
    )
}
