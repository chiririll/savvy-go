import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { testId } from '@/lib/test-id'

export function BackButton({ visible, onClick }: { visible: boolean; onClick: () => void }) {
    const { t } = useTranslation('common')
    if (!visible) return null

    return (
        <Button type="button" variant="outline" onClick={onClick} {...testId('create-space-back')}>
            {t('actions.back')}
        </Button>
    )
}

export function CancelButton({ visible, onClick }: { visible: boolean; onClick: () => void }) {
    const { t } = useTranslation('common')
    if (!visible) return null

    return (
        <Button type="button" variant="ghost" onClick={onClick}>
            {t('actions.cancel')}
        </Button>
    )
}

export function NextButton({ disabled, onClick }: { disabled: boolean; onClick: () => void }) {
    const { t } = useTranslation('common')

    return (
        <Button type="button" disabled={disabled} onClick={onClick} {...testId('create-space-next')}>
            {t('actions.continue')}
        </Button>
    )
}
