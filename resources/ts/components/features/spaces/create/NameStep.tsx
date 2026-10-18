import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface Props {
    name: string
    onNameChange: (name: string) => void
    /** Enter in the field; the caller decides whether the name is good enough to go on. */
    onSubmit: () => void
}

export function NameStep({ name, onNameChange, onSubmit }: Props) {
    const { t } = useTranslation('settings')

    return (
        <div className="space-y-5">
            <ul className="list-disc space-y-1.5 pl-5 text-sm text-muted-foreground">
                <li>{t('spaces.create.intro.finances')}</li>
                <li>{t('spaces.create.intro.currency')}</li>
                <li>{t('spaces.create.intro.members')}</li>
                <li>{t('spaces.create.intro.separate')}</li>
            </ul>
            <div className="space-y-2">
                <Label htmlFor="space-name">{t('spaces.fields.name')}</Label>
                <Input
                    id="space-name"
                    value={name}
                    maxLength={100}
                    autoFocus
                    onChange={(e) => onNameChange(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && name.trim() && onSubmit()}
                />
            </div>
        </div>
    )
}
