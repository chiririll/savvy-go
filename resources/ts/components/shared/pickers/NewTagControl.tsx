import { useState } from 'react'
import { Check, Plus, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { useCreateTag, useTags } from '@/hooks'
import { getApiErrorMessage } from '@/lib/api-error'
import { tagSchema } from '@/schemas/tags'

interface NewTagControlProps {
    value: number[]
    onChange: (value: number[]) => void
}

const iconButtonClass = 'flex size-6 items-center justify-center rounded-md border hover:bg-muted'

/** "+ New tag" chip that expands into an input with confirm and cancel buttons. */
export function NewTagControl({ value, onChange }: NewTagControlProps) {
    const { t } = useTranslation('forms')
    const { data: tags } = useTags()
    const createTag = useCreateTag()
    const [creating, setCreating] = useState(false)
    const [name, setName] = useState('')
    const [error, setError] = useState<string>()

    const close = () => {
        setCreating(false)
        setName('')
        setError(undefined)
    }

    const submit = async () => {
        const trimmed = name.trim().replace(/^#/, '').trim()
        if (!trimmed) {
            close()
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
            close()
            return
        }

        try {
            const created = await createTag.mutateAsync(parsed.data)
            onChange([...value, created.id])
            close()
        } catch (e) {
            toast.error(getApiErrorMessage(e, t('tags.createFailed')))
        }
    }

    if (!creating) {
        return (
            <Badge asChild variant="outline" className="cursor-pointer border-dashed hover:bg-muted">
                <button type="button" onClick={() => setCreating(true)}>
                    <Plus className="size-3" />
                    {t('tags.new')}
                </button>
            </Badge>
        )
    }

    return (
        <div className="flex flex-col gap-1">
            <div className="flex items-center gap-1">
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
                            close()
                        }
                    }}
                    onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                            // Don't submit the surrounding transaction form.
                            e.preventDefault()
                            e.stopPropagation()
                            void submit()
                        } else if (e.key === 'Escape') {
                            e.preventDefault()
                            e.stopPropagation()
                            close()
                        }
                    }}
                />
                {/* mouseDown preventDefault keeps input focus, so the blur handler can't swallow the tap on touch screens. */}
                <button
                    type="button"
                    aria-label={t('tags.confirm')}
                    disabled={createTag.isPending}
                    className={iconButtonClass}
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={() => void submit()}
                >
                    <Check className="size-3.5" />
                </button>
                <button
                    type="button"
                    aria-label={t('tags.cancel')}
                    className={iconButtonClass}
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={close}
                >
                    <X className="size-3.5" />
                </button>
            </div>
            {error && <span className="text-xs text-destructive">{error}</span>}
        </div>
    )
}
