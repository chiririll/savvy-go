import { useState } from 'react'
import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { FormDescription, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { useCreateTag, useTags } from '@/hooks'
import { getApiErrorMessage } from '@/lib/api-error'
import { cn } from '@/lib/utils'
import { localizeDefaultName } from '@/lib/localized-name'
import { tagSchema } from '@/schemas/tags'

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
    const createTag = useCreateTag()
    const [creating, setCreating] = useState(false)
    const [name, setName] = useState('')
    const [error, setError] = useState<string>()

    if (!tags?.length && !allowCreate) {
        return null
    }

    const closeCreate = () => {
        setCreating(false)
        setName('')
        setError(undefined)
    }

    const submitCreate = async () => {
        const trimmed = name.trim().replace(/^#/, '')
        if (!trimmed) {
            closeCreate()
            return
        }

        const parsed = tagSchema.safeParse({ name: trimmed })
        if (!parsed.success) {
            setError(parsed.error.issues[0]?.message)
            return
        }

        const existing = tags?.find((tag) => tag.name.toLowerCase() === trimmed.toLowerCase())
        if (existing) {
            if (!value.includes(existing.id)) {
                onChange([...value, existing.id])
            }
            closeCreate()
            return
        }

        try {
            const created = await createTag.mutateAsync(parsed.data)
            onChange([...value, created.id])
            closeCreate()
        } catch (e) {
            toast.error(getApiErrorMessage(e, t('tags.createFailed')))
        }
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
            {allowCreate &&
                (creating ? (
                    <div className="flex flex-col gap-1">
                        <Input
                            autoFocus
                            value={name}
                            disabled={createTag.isPending}
                            placeholder={t('tags.namePlaceholder')}
                            aria-label={t('tags.newName')}
                            aria-invalid={!!error}
                            className="h-6 w-36 px-2 text-xs"
                            onChange={(e) => {
                                setName(e.target.value)
                                setError(undefined)
                            }}
                            onBlur={() => {
                                if (!name.trim()) {
                                    closeCreate()
                                }
                            }}
                            onKeyDown={(e) => {
                                if (e.key === 'Enter') {
                                    // Don't submit the surrounding transaction form.
                                    e.preventDefault()
                                    e.stopPropagation()
                                    void submitCreate()
                                } else if (e.key === 'Escape') {
                                    e.preventDefault()
                                    e.stopPropagation()
                                    closeCreate()
                                }
                            }}
                        />
                        {error && <span className="text-xs text-destructive">{error}</span>}
                    </div>
                ) : (
                    <Badge asChild variant="outline" className="cursor-pointer border-dashed hover:bg-muted">
                        <button type="button" onClick={() => setCreating(true)}>
                            <Plus className="size-3" />
                            {t('tags.new')}
                        </button>
                    </Badge>
                ))}
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
