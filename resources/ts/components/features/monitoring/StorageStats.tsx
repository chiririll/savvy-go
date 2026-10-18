import { Boxes, Database, FileSpreadsheet, UploadCloud } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { StorageSnapshot } from '@/types/monitoring'

import { formatBytes, formatCount } from './format'
import { StatCard } from './parts'

export function StorageStats({ data }: { data: StorageSnapshot }) {
    const { t } = useTranslation('settings')
    const { managed, uploads, imports } = data

    return (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard
                icon={Database}
                label={t('monitoring.savvyData')}
                value={formatBytes(managed.used_bytes)}
                hint={t('monitoring.storedObjects', { count: managed.objects })}
            />
            <StatCard
                icon={UploadCloud}
                label={t('monitoring.inFlight')}
                value={formatBytes(managed.pending_bytes)}
                hint={t('monitoring.pending', { count: uploads.by_status.pending ?? 0 })}
            />
            <StatCard
                icon={Boxes}
                label={t('monitoring.uploads')}
                value={formatCount(uploads.total)}
                hint={t('monitoring.completed', { count: uploads.by_status.completed ?? 0 })}
            />
            <StatCard
                icon={FileSpreadsheet}
                label={t('monitoring.imports')}
                value={formatCount(imports.total)}
                hint={t('monitoring.completed', { count: imports.by_status.completed ?? 0 })}
            />
        </div>
    )
}
