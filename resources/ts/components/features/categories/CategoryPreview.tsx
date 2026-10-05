import { useTranslation } from 'react-i18next'
import { FeedRow } from '@/components/shared'
import { categoryIconStyle } from '@/lib/category-color'
import { localizeDefaultName } from '@/lib/localized-name'

interface CategoryPreviewProps {
    name: string
    icon: string
    color: string
}

/** The category as lists will show it. */
export function CategoryPreview({ name, icon, color }: CategoryPreviewProps) {
    const { t } = useTranslation('forms')

    return (
        <div className="rounded-lg border bg-muted/50 px-4 pb-2 pt-3">
            <p className="mb-1 text-sm text-muted-foreground">{t('categories.preview')}</p>
            <FeedRow
                icon={<span aria-hidden>{icon}</span>}
                iconStyle={categoryIconStyle(color)}
                title={localizeDefaultName(name) || t('categories.previewName')}
            />
        </div>
    )
}
