import { Boxes } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { StorageSnapshot } from '@/types/monitoring'

import { formatBytes } from './format'
import { PanelHeader, PanelSubtitle } from './parts'

export function BucketBreakdown({ data }: { data: StorageSnapshot }) {
    const { t } = useTranslation('settings')
    const { buckets, used_bytes } = data.managed

    return (
        <section className="rounded-2xl border bg-card shadow-sm">
            <PanelHeader
                icon={Boxes}
                title={t('monitoring.storageByBucket')}
                subtitle={<PanelSubtitle>{t('monitoring.storageByBucketDescription')}</PanelSubtitle>}
            />

            {buckets.length === 0 ? (
                <p className="px-6 py-10 text-center text-sm text-muted-foreground">{t('monitoring.nothingStored')}</p>
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
