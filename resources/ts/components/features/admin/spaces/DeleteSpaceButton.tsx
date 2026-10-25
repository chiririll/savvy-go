import { useTranslation } from 'react-i18next'
import { Trash2 } from 'lucide-react'
import { Confirm } from '@/components/shared/dialogs/Confirm'
import { Button } from '@/components/ui/button'
import { useAdminDeleteSpace } from '@/hooks'
import type { SpaceOverview } from '@/types/spaces'

export function DeleteSpaceButton({ space }: { space: SpaceOverview }) {
    const { t } = useTranslation('settings')
    const remove = useAdminDeleteSpace()

    return (
        <Confirm
            trigger={
                <Button variant="ghost" size="icon" title={t('actions.delete', { ns: 'common' })}>
                    <Trash2 className="size-4" />
                </Button>
            }
            title={t('spaces.general.deleteTitle', { name: space.name })}
            description={t('spaces.general.deleteDescription')}
            action={t('actions.delete', { ns: 'common' })}
            onConfirm={() => remove.mutate(space.id)}
        />
    )
}
