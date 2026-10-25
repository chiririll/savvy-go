import { useTranslation } from 'react-i18next'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { Space } from '@/types/spaces'
import { COPY_KINDS, type CopyKind, type CopyKinds } from './draft'

interface Props {
    spaces: Space[]
    /** The space to copy from; null for the built-in defaults. */
    sourceId: number | null
    copy: CopyKinds
    onSourceChange: (sourceId: number | null) => void
    onCopyChange: (copy: CopyKinds) => void
}

export function CopyStep({ spaces, sourceId, copy, onSourceChange, onCopyChange }: Props) {
    const { t } = useTranslation('settings')
    const kindLabels: Record<CopyKind, string> = {
        currencies: t('spaces.create.copy.kinds.currencies'),
        accounts: t('spaces.create.copy.kinds.accounts'),
        categories: t('spaces.create.copy.kinds.categories'),
        tags: t('spaces.create.copy.kinds.tags'),
    }

    return (
        <div className="space-y-4">
            <div className="space-y-1.5">
                <Label htmlFor="create-space-source">{t('spaces.create.copy.source')}</Label>
                <Select
                    value={sourceId === null ? 'defaults' : String(sourceId)}
                    onValueChange={(v) => onSourceChange(v === 'defaults' ? null : Number(v))}
                >
                    <SelectTrigger id="create-space-source" className="w-full">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="defaults">{t('spaces.create.copy.defaults')}</SelectItem>
                        {spaces.map((space) => (
                            <SelectItem key={space.id} value={String(space.id)}>
                                {space.name}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            </div>
            {sourceId !== null && (
                <fieldset className="space-y-2">
                    <legend className="text-sm font-medium">{t('spaces.create.copy.what')}</legend>
                    {COPY_KINDS.map((kind) => (
                        <label key={kind} className="flex items-center gap-2 text-sm">
                            <Checkbox
                                checked={copy[kind]}
                                onCheckedChange={(checked) => onCopyChange({ ...copy, [kind]: checked === true })}
                            />
                            {kindLabels[kind]}
                        </label>
                    ))}
                    <p className="text-xs text-muted-foreground">{t('spaces.create.copy.hint')}</p>
                </fieldset>
            )}
        </div>
    )
}
