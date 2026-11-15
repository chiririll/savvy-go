import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { testId } from '@/lib/test-id'

interface CreateSubmitButtonProps {
    disabled: boolean
    pending: boolean
    onClick: () => void
}

export function CreateSubmitButton({ disabled, pending, onClick }: CreateSubmitButtonProps) {
    const { t } = useTranslation('settings')
    return (
        <Button type="button" disabled={disabled || pending} onClick={onClick} {...testId('create-space-submit')}>
            {pending && <Loader2 className="size-4 animate-spin" />}
            {t('spaces.create.submit')}
        </Button>
    )
}
