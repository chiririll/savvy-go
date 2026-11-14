import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { FormDescription, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { useTags } from '@/hooks'
import { cn } from '@/lib/utils'
import { localizeDefaultName } from '@/lib/localized-name'
import { NewTagControl } from './NewTagControl'

interface TagSelectProps {
    value?: number[]
    onChange: (value: number[]) => void
    bordered?: boolean
    asFormItem?: boolean
    label?: string
    description?: string
    /** Show an inline control to create a tag and select it right away. */
    allowCreate?: boolean
}

export function TagSelect({
    value = [],
    onChange,
    bordered,
    asFormItem,
    label,
    description,
    allowCreate,
}: TagSelectProps) {
    const { t } = useTranslation('forms')
    const { data: tags } = useTags()

    if (!tags?.length && !allowCreate) {
        return null
    }

    const chips = (
        <div className={cn('flex flex-wrap items-center gap-2', bordered && 'p-3 rounded-md border')}>
            {tags?.map((tag) => {
                const isSelected = value.includes(tag.id)

                return (
                    <Badge
                        key={tag.id}
                        asChild
                        variant={isSelected ? 'default' : 'outline'}
                        className={cn(
                            'cursor-pointer transition-colors',
                            isSelected ? 'hover:bg-primary/80' : 'hover:bg-muted',
                        )}
                    >
                        <button
                            type="button"
                            aria-pressed={isSelected}
                            onClick={() => {
                                onChange(isSelected ? value.filter((id) => id !== tag.id) : [...value, tag.id])
                            }}
                        >
                            #{localizeDefaultName(tag.name, 'tags')}
                        </button>
                    </Badge>
                )
            })}
            {allowCreate && <NewTagControl value={value} onChange={onChange} />}
        </div>
    )

    if (!asFormItem) {
        return chips
    }

    return (
        <FormItem>
            <FormLabel>{label ?? t('tags.label')}</FormLabel>
            {description && <FormDescription>{description}</FormDescription>}
            {chips}
            <FormMessage />
        </FormItem>
    )
}
