import { useTranslation } from 'react-i18next'
import { History } from 'lucide-react'
import { SettingsSection } from '@/components/shared'
import { useSpaceAudit } from '@/hooks'
import { formatDateTime } from '@/lib/format'
import { Space } from '@/types/spaces'
import { auditActionLabel } from '@/lib/labels'

export function Audit({ space }: { space: Space }) {
    const { t } = useTranslation('settings')
    const { data: entries = [] } = useSpaceAudit(space.id)
    return (
        <SettingsSection icon={History} title={t('spaces.audit.title')} description={t('spaces.audit.description')}>
            {entries.length ? (
                <ul className="divide-y rounded-lg border text-sm">
                    {entries.map((e) => (
                        <li key={e.id} className="flex justify-between gap-3 px-3 py-2">
                            <span>
                                {e.actor || t('spaces.audit.someone')} — {auditActionLabel(t, e.action)}
                                {e.details ? ` (${e.details})` : ''}
                            </span>
                            <span className="shrink-0 text-xs text-muted-foreground">
                                {formatDateTime(e.createdAt)}
                            </span>
                        </li>
                    ))}
                </ul>
            ) : (
                <p className="text-sm text-muted-foreground">{t('spaces.audit.none')}</p>
            )}
        </SettingsSection>
    )
}
