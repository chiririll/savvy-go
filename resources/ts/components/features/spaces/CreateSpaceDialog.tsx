import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog'
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
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-md">
                <form onSubmit={submit} className="space-y-4">
                    <DialogHeader>
                        <DialogTitle>{t('spaces.create.title')}</DialogTitle>
                        <DialogDescription>{t('spaces.create.description')}</DialogDescription>
                    </DialogHeader>
                    <div className="space-y-2">
                        <Label htmlFor="space-name">{t('spaces.fields.name')}</Label>
                        <Input id="space-name" value={name} maxLength={100} autoFocus onChange={(e) => setName(e.target.value)} />
                    </div>
                    <DialogFooter>
                        <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                            {t('actions.cancel', { ns: 'common' })}
                        </Button>
                        <Button type="submit" disabled={!name.trim() || create.isPending}>
                            {t('spaces.create.submit')}
                        </Button>
                    </DialogFooter>
                </form>
            </DialogContent>
        </Dialog>
    )
}
