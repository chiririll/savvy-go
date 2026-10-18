import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { numberDraft, numberToDraft } from './setting-drafts'

/** A number field saved on demand; empty means null when nullable. */
export function NumberRow({
    id,
    label,
    description,
    value,
    nullable,
    onSave,
}: {
    id: string
    label: string
    description: string
    value: number | null
    nullable?: boolean
    onSave: (value: number | null) => void
}) {
    const { t } = useTranslation('settings')
    const [draft, setDraft] = useState(numberToDraft(value))
    useEffect(() => setDraft(numberToDraft(value)), [value])
    const { parsed, valid, changed } = numberDraft(draft, value, nullable)

    return (
        <div className="flex items-center justify-between gap-4 px-4 py-3.5">
            <div className="space-y-1">
                <Label htmlFor={id} className="text-sm font-medium">
                    {label}
                </Label>
                <p className="text-sm leading-relaxed text-muted-foreground">{description}</p>
            </div>
            <div className="flex items-center gap-2">
                <Input
                    id={id}
                    type="number"
                    min={0}
                    step="any"
                    className="w-28"
                    value={draft}
                    onChange={(e) => setDraft(e.target.value)}
                />
                {changed && (
                    <Button size="sm" disabled={!valid} onClick={() => onSave(parsed)}>
                        {t('actions.save', { ns: 'common' })}
                    </Button>
                )}
            </div>
        </div>
    )
}

/** A text field saved on demand; empty clears the setting. */
export function TextRow({
    id,
    label,
    description,
    placeholder,
    value,
    readOnly,
    onSave,
}: {
    id: string
    label: string
    description: string
    placeholder?: string
    value: string
    readOnly?: boolean
    onSave: (value: string) => void
}) {
    const { t } = useTranslation('settings')
    const [draft, setDraft] = useState(value)
    useEffect(() => setDraft(value), [value])
    const changed = draft.trim() !== value

    return (
        <div className="flex flex-col gap-3 px-4 py-3.5 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-1">
                <Label htmlFor={id} className="text-sm font-medium">
                    {label}
                </Label>
                <p className="text-sm leading-relaxed text-muted-foreground">{description}</p>
            </div>
            <div className="flex items-center gap-2">
                <Input
                    id={id}
                    type="url"
                    className="w-full sm:w-72"
                    placeholder={placeholder}
                    value={draft}
                    readOnly={readOnly}
                    onChange={(e) => setDraft(e.target.value)}
                />
                {changed && !readOnly && (
                    <Button size="sm" onClick={() => onSave(draft.trim())}>
                        {t('actions.save', { ns: 'common' })}
                    </Button>
                )}
            </div>
        </div>
    )
}
