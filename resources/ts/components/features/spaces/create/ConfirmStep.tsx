import { useTranslation } from 'react-i18next'
import type { toRequest } from './draft'

export function ConfirmStep({ request }: { request: ReturnType<typeof toRequest> }) {
    const { t } = useTranslation('settings')

    return (
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
            <dt className="text-muted-foreground">{t('spaces.fields.name')}</dt>
            <dd>{request.name}</dd>
            <dt className="text-muted-foreground">{t('spaces.create.summary.base')}</dt>
            <dd>{request.baseCurrency}</dd>
            <dt className="text-muted-foreground">{t('spaces.create.summary.currencies')}</dt>
            <dd>{request.currencies?.length ? request.currencies.join(', ') : '—'}</dd>
            <dt className="text-muted-foreground">{t('spaces.create.summary.accounts')}</dt>
            <dd>{request.accounts?.length ?? 0}</dd>
            <dt className="text-muted-foreground">{t('spaces.create.summary.categories')}</dt>
            <dd>{request.categories?.length ?? 0}</dd>
            <dt className="text-muted-foreground">{t('spaces.create.summary.tags')}</dt>
            <dd>{request.tags?.length ?? 0}</dd>
        </dl>
    )
}
