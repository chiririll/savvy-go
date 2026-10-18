import { RefreshCw } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { BucketBreakdown } from '@/components/features/monitoring/BucketBreakdown'
import { ResourcePanel } from '@/components/features/monitoring/ResourcePanel'
import { StorageStats } from '@/components/features/monitoring/StorageStats'
import { VolumePanel } from '@/components/features/monitoring/VolumePanel'
import { Page, PageHeader } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useResourceMonitoring, useStorageMonitoring } from '@/hooks'
import { intlLocale } from '@/lib/i18n'
import { cn } from '@/lib/utils'

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

function LiveIndicator({ time }: { time: string }) {
    const { t } = useTranslation('settings')
    return (
        <span className="hidden items-center gap-1.5 text-xs text-muted-foreground sm:flex">
            <span className="relative flex size-2">
                <span className="absolute inline-flex size-full animate-ping rounded-full bg-emerald-500 opacity-60" />
                <span className="relative inline-flex size-2 rounded-full bg-emerald-500" />
            </span>
            {t('monitoring.updated', { time })}
        </span>
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
                        {updatedLabel && <LiveIndicator time={updatedLabel} />}
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
                            <StorageStats data={storage.data} />
                            <BucketBreakdown data={storage.data} />
                        </>
                    )}
                </div>
            )}
        </Page>
    )
}
