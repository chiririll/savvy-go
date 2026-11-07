import { Pencil } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { MarkdownField } from '@/components/shared/markdown/MarkdownField'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'

interface LineNoteFieldProps {
    value: string
    onChange: (value: string) => void
    onBlur?: () => void
    /** Names the field for assistive technology; the note has no visible label. */
    label: string
}

/**
 * The note of a line: a compact input that grows with the text, and a pencil that opens the markdown editor
 * in its place. A note that was saved opens in the editor, so its formatting is not hidden.
 */
export function LineNoteField({ value, onChange, onBlur, label }: LineNoteFieldProps) {
    const { t } = useTranslation('forms')
    const [editing, setEditing] = useState(() => value.trim() !== '')

    if (editing) {
        return (
            <MarkdownField
                value={value}
                onChange={onChange}
                onBlur={onBlur}
                rows={2}
                placeholder={t('transactions.lineNotePlaceholder')}
                onClose={() => setEditing(false)}
            />
        )
    }

    return (
        <div className="flex items-start gap-1">
            <Textarea
                rows={1}
                value={value}
                aria-label={label}
                placeholder={t('transactions.lineNotePlaceholder')}
                className="min-h-9 resize-none py-1.5"
                onChange={(e) => onChange(e.target.value)}
                onBlur={onBlur}
            />
            <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label={t('transactions.editNote')}
                title={t('transactions.editNote')}
                onClick={() => setEditing(true)}
            >
                <Pencil />
            </Button>
        </div>
    )
}
