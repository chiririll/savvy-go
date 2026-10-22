import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { FormDescription, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { useTags } from '@/hooks'
import { cn } from '@/lib/utils'
import { localizeDefaultName } from '@/lib/localized-name'

interface TagSelectProps {
    value?: number[]
    onChange: (value: number[]) => void
    bordered?: boolean
    asFormItem?: boolean
    label?: string
    description?: string
}

export function TagSelect({
    value = [],
    onChange,
    bordered,
    asFormItem,
    label,
    description,
}: TagSelectProps) {
    const { t } = useTranslation('forms')
    const { data: tags } = useTags()

    if (!tags?.length) {
        return null
    }

    const chips = (
        <div className={cn('flex flex-wrap gap-2', bordered && 'p-3 rounded-md border')}>
            {tags.map((tag) => {
                const isSelected = value.includes(tag.id)

                return (
                    <Badge
                        key={tag.id}
                        asChild
                        variant={isSelected ? 'default' : 'outline'}
                        className={cn(
                            'cursor-pointer transition-colors',
                            isSelected ? 'hover:bg-primary/80' : 'hover:bg-muted'
                        )}
                    >
                        <button
                            type="button"
                            aria-pressed={isSelected}
                            onClick={() => {
                                onChange(
                                    isSelected
                                        ? value.filter((id) => id !== tag.id)
                                        : [...value, tag.id]
                                )
                            }}
                        >
                            #{localizeDefaultName(tag.name, 'tags')}
                        </button>
                    </Badge>
                )
            })}
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
