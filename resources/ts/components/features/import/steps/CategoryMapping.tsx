import { useTranslation } from 'react-i18next'
import { transactionTypeLabelLoose } from '@/lib/labels'
import { Badge } from '@/components/ui/badge'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import { useCategories } from '@/hooks'
import { intlLocale } from '@/lib/i18n'
import type { ImportCategoryCandidate, ImportCategoryChoice, ImportCategoryMap } from '@/types/import'

interface CategoryMappingProps {
    candidates: ImportCategoryCandidate[]
    value: ImportCategoryMap
    onChange: (value: ImportCategoryMap) => void
    disabled?: boolean
}

const toSelectValue = (choice: ImportCategoryChoice | undefined): string =>
    choice === undefined ? 'create' : String(choice)

const fromSelectValue = (value: string): ImportCategoryChoice =>
    value === 'create' || value === 'skip' ? value : Number(value)

export function CategoryMapping({ candidates, value, onChange, disabled }: CategoryMappingProps) {
    const { t } = useTranslation('settings')
    const { t: tPages } = useTranslation('pages')
    const { data: categories = [] } = useCategories()

    if (candidates.length === 0) {
        return null
    }

    const toCreate = candidates.filter((c) => value[c.name] === 'create').length

    return (
        <div className="space-y-3 rounded-lg border p-4">
            <div>
                <div className="font-medium">{t('import.categoryMapping.title')}</div>
                <p className="text-sm text-muted-foreground">
                    {t('import.categoryMapping.description')}
                    {toCreate > 0 && ` ${t('import.categoryMapping.willCreate', { count: toCreate })}`}
                </p>
            </div>

            <div className="divide-y">
                {candidates.map((candidate) => {
                    const options = categories.filter((c) => c.type === candidate.type)

                    return (
                        <div
                            key={candidate.name}
                            className="grid items-center gap-2 py-2 sm:grid-cols-[1fr_minmax(0,16rem)]"
                        >
                            <div className="flex min-w-0 items-center gap-2">
                                <span className="truncate font-medium">{candidate.name}</span>
                                <Badge variant="secondary">
                                    {transactionTypeLabelLoose(tPages, candidate.type)}
                                </Badge>
                                <span className="shrink-0 text-xs text-muted-foreground">
                                    {t('import.categoryMapping.rows', { count: candidate.count.toLocaleString(intlLocale()) })}
                                </span>
                            </div>
                            <Select
                                value={toSelectValue(value[candidate.name])}
                                onValueChange={(next) => onChange({ ...value, [candidate.name]: fromSelectValue(next) })}
                                disabled={disabled}
                            >
                                <SelectTrigger className="w-full">
                                    <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="create">
                                        {t('import.categoryMapping.create', { name: candidate.name })}
                                    </SelectItem>
                                    <SelectItem value="skip">{t('import.categoryMapping.skip')}</SelectItem>
                                    {options.map((category) => (
                                        <SelectItem key={category.id} value={String(category.id)}>
                                            {category.name}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                        </div>
                    )
                })}
            </div>
        </div>
    )
}
