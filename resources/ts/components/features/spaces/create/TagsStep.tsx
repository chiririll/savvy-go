import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { DialogPinned } from '@/components/shared/dialogs/DialogPinned'
import { localizeDefaultName } from '@/lib/localized-name'
import type { DraftContent } from './draft'
import { StepSection } from './StepSection'

interface Props {
    content: DraftContent
    onChange: (patch: Partial<DraftContent>) => void
}

/** Step: the tags of the space. */
export function TagsStep({ content, onChange }: Props) {
    const { t } = useTranslation('settings')
    const [name, setName] = useState('')

    return (
        <StepSection title={t('spaces.create.tags.title')} hint={t('spaces.create.tags.hint')}>
            <div className="flex flex-wrap gap-2">
                {content.tags.map((tag) => (
                    <span key={tag} className="inline-flex items-center gap-1 rounded-md border px-2 py-1 text-sm">
                        {localizeDefaultName(tag, 'tags')}
                        <button
                            type="button"
                            aria-label={t('spaces.create.tags.remove', { name: localizeDefaultName(tag, 'tags') })}
                            className="text-muted-foreground hover:text-foreground"
                            onClick={() => onChange({ tags: content.tags.filter((x) => x !== tag) })}
                        >
                            <X className="size-3.5" />
                        </button>
                    </span>
                ))}
            </div>
            <DialogPinned>
            <form
                className="flex gap-2"
                data-testid="create-space-tag-form"
                onSubmit={(e) => {
                    e.preventDefault()
                    const tag = name.trim()
                    if (tag && !content.tags.some((x) => x.toLowerCase() === tag.toLowerCase()))
                        onChange({ tags: [...content.tags, tag] })
                    setName('')
                }}
            >
                <Input
                    aria-label={t('spaces.create.tags.new')}
                    placeholder={t('spaces.create.tags.new')}
                    value={name}
                    maxLength={50}
                    onChange={(e) => setName(e.target.value)}
                />
                <Button type="submit" variant="outline" size="icon" aria-label={t('spaces.create.tags.add')}>
                    <Plus className="size-4" />
                </Button>
            </form>
            </DialogPinned>
        </StepSection>
    )
}
