import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { categoryTypeLabel } from '@/lib/labels'
import { defaultNameHint, localizeDefaultName, toStoredDefaultName } from '@/lib/localized-name'
import { CATEGORY_TYPES, type CategoryType } from '@/types/categories'
import { isProtectedCategory, removeCategory, type DraftContent } from './draft'
import { StepSection } from './StepSection'

interface Props {
    content: DraftContent
    onChange: (patch: Partial<DraftContent>) => void
}

/** Step: the categories of the space. */
export function CategoriesStep({ content, onChange }: Props) {
    const { t } = useTranslation('settings')
    const { t: tc } = useTranslation()
    const [name, setName] = useState('')
    const [type, setType] = useState<CategoryType>('expense')
    const rename = (index: number, input: string) =>
        onChange({
            categories: content.categories.map((c, i) =>
                i === index ? { ...c, name: toStoredDefaultName(input, c.name) } : c,
            ),
        })

    return (
        <StepSection title={t('spaces.create.categories.title')} hint={t('spaces.create.categories.hint')}>
            <ul className="space-y-1.5">
                {content.categories.map((category, index) => (
                    <li key={index} className="flex items-center gap-2">
                        <span aria-hidden className="w-6 text-center">
                            {category.icon}
                        </span>
                        <div className="min-w-0 flex-1">
                            <Input
                                aria-label={t('spaces.create.categories.name')}
                                value={localizeDefaultName(category.name)}
                                maxLength={255}
                                onChange={(e) => rename(index, e.target.value)}
                            />
                            {defaultNameHint(category.name) && (
                                <p className="mt-0.5 truncate text-xs text-muted-foreground">
                                    {defaultNameHint(category.name)}
                                </p>
                            )}
                        </div>
                        <span className="w-20 shrink-0 text-xs text-muted-foreground">
                            {categoryTypeLabel(tc, category.type)}
                        </span>
                        <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            disabled={isProtectedCategory(category)}
                            title={isProtectedCategory(category) ? t('spaces.create.categories.protected') : undefined}
                            aria-label={t('spaces.create.categories.remove', {
                                name: localizeDefaultName(category.name),
                            })}
                            onClick={() => onChange({ categories: removeCategory(content.categories, index) })}
                        >
                            <X className="size-4" />
                        </Button>
                    </li>
                ))}
            </ul>
            <form
                className="flex gap-2"
                onSubmit={(e) => {
                    e.preventDefault()
                    if (!name.trim()) return
                    onChange({
                        categories: [
                            ...content.categories,
                            { name: name.trim(), type, icon: '🏷️', color: '#6b7280', isDefault: false },
                        ],
                    })
                    setName('')
                }}
            >
                <Input
                    aria-label={t('spaces.create.categories.new')}
                    placeholder={t('spaces.create.categories.new')}
                    value={name}
                    maxLength={255}
                    onChange={(e) => setName(e.target.value)}
                />
                <Select value={type} onValueChange={(v) => setType(v as CategoryType)}>
                    <SelectTrigger className="w-32" aria-label={tc('fields.type')}>
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        {CATEGORY_TYPES.map((v) => (
                            <SelectItem key={v} value={v}>
                                {categoryTypeLabel(tc, v)}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
                <Button type="submit" variant="outline" size="icon" aria-label={t('spaces.create.categories.add')}>
                    <Plus className="size-4" />
                </Button>
            </form>
        </StepSection>
    )
}
