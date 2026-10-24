import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowDownLeft, ArrowUpRight, Plus, X } from 'lucide-react'
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

/** The same arrows as the transaction list uses for the two kinds of money flow. */
const TYPE_ICONS = {
    expense: { Icon: ArrowUpRight, tone: 'text-destructive' },
    income: { Icon: ArrowDownLeft, tone: 'text-success' },
} satisfies Record<CategoryType, { Icon: typeof ArrowUpRight; tone: string }>

/** Step: the categories of the space, in a group per type, and a form to add one that stays in view. */
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

    const groups: { type: CategoryType; title: string }[] = [
        { type: 'expense', title: t('spaces.create.categories.groupExpense') },
        { type: 'income', title: t('spaces.create.categories.groupIncome') },
    ]

    return (
        <StepSection title={t('spaces.create.categories.title')} hint={t('spaces.create.categories.hint')}>
            {groups.map((group) => {
                // Indexes stay those of the whole list: renaming and removing address it by position.
                const rows = content.categories
                    .map((category, index) => ({ category, index }))
                    .filter(({ category }) => category.type === group.type)
                if (rows.length === 0) return null
                return (
                    <div key={group.type} className="space-y-1.5" data-testid={`create-space-categories-${group.type}`}>
                        <h4 className="pt-1 text-xs font-medium text-muted-foreground">{group.title}</h4>
                        <ul className="space-y-1.5">
                            {rows.map(({ category, index }) => (
                                // Top-aligned: the emoji and the button line up with the input, whatever the hint below it adds.
                                <li key={index} className="flex items-start gap-2">
                                    <span aria-hidden className="flex h-9 w-6 shrink-0 items-center justify-center">
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
                                    <Button
                                        type="button"
                                        variant="ghost"
                                        size="icon"
                                        className="shrink-0"
                                        disabled={isProtectedCategory(category)}
                                        title={
                                            isProtectedCategory(category)
                                                ? t('spaces.create.categories.protected')
                                                : undefined
                                        }
                                        aria-label={t('spaces.create.categories.remove', {
                                            name: localizeDefaultName(category.name),
                                        })}
                                        onClick={() =>
                                            onChange({ categories: removeCategory(content.categories, index) })
                                        }
                                    >
                                        <X className="size-4" />
                                    </Button>
                                </li>
                            ))}
                        </ul>
                    </div>
                )
            })}
            {/* Sticks to the bottom of the modal's scroll area, so it is there however long the list is.
                The negative margins stretch it over the area's padding; it is a form, so the keyboard
                of a phone shrinks the sheet (dvh) and this stays above it. */}
            <form
                className="sticky bottom-0 z-10 -mx-6 -mb-4 flex gap-2 border-t bg-background px-6 py-3"
                data-testid="create-space-category-form"
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
                    {/* Phone: the closed select shows the icon only; the opened list has icon and name. */}
                    <SelectTrigger
                        className="w-16 shrink-0 sm:w-36 max-sm:[&_[data-type-name]]:hidden"
                        aria-label={tc('fields.type')}
                    >
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        {CATEGORY_TYPES.map((v) => {
                            const { Icon, tone } = TYPE_ICONS[v]
                            return (
                                <SelectItem key={v} value={v}>
                                    <Icon aria-hidden className={tone} />
                                    <span data-type-name>{categoryTypeLabel(tc, v)}</span>
                                </SelectItem>
                            )
                        })}
                    </SelectContent>
                </Select>
                <Button type="submit" variant="outline" size="icon" className="shrink-0" aria-label={t('spaces.create.categories.add')}>
                    <Plus className="size-4" />
                </Button>
            </form>
        </StepSection>
    )
}
