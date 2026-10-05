import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { ResponsiveDialog } from '@/components/shared/ResponsiveDialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useCreateSpace, useSwitchSpace } from '@/hooks/use-spaces'

export function CreateSpaceDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
    const { t } = useTranslation('settings')
    const [name, setName] = useState('')
    const create = useCreateSpace()
    const switchSpace = useSwitchSpace()

    const submit = async (event: React.FormEvent) => {
        event.preventDefault()
        const space = await create.mutateAsync(name.trim())
        setName('')
        onOpenChange(false)
        await switchSpace(space.id)
    }

    return (
        <ResponsiveDialog
            open={open}
            onOpenChange={onOpenChange}
            title={t('spaces.create.title')}
            description={t('spaces.create.description')}
            footer={
                <>
                    <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                        {t('actions.cancel', { ns: 'common' })}
                    </Button>
                    <Button type="submit" form="create-space-form" disabled={!name.trim() || create.isPending}>
                        {t('spaces.create.submit')}
                    </Button>
                </>
            }
        >
            <form id="create-space-form" onSubmit={submit} className="space-y-2">
                <Label htmlFor="space-name">{t('spaces.fields.name')}</Label>
                <Input id="space-name" value={name} maxLength={100} autoFocus onChange={(e) => setName(e.target.value)} />
            </form>
        </ResponsiveDialog>
    )
}
