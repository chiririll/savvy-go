import { testId } from '@/lib/test-id'
import { useTranslation } from 'react-i18next'
import { Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { useAdminDeleteSpace } from '@/hooks'
import type { SpaceOverview } from '@/types/spaces'

export function DeleteSpaceButton({ space }: { space: SpaceOverview }) {
    const { t } = useTranslation('settings')
    const remove = useAdminDeleteSpace()

    return (
        <AlertDialog>
            <AlertDialogTrigger asChild {...testId('confirm')}>
                <Button variant="ghost" size="icon" title={t('actions.delete', { ns: 'common' })}>
                    <Trash2 className="size-4" />
                </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>{t('spaces.general.deleteTitle', { name: space.name })}</AlertDialogTitle>
                    <AlertDialogDescription>{t('spaces.general.deleteDescription')}</AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel>{t('actions.cancel', { ns: 'common' })}</AlertDialogCancel>
                    <AlertDialogAction onClick={() => remove.mutate(space.id)}>
                        {t('actions.delete', { ns: 'common' })}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    )
}
