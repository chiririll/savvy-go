import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'

interface FormSubmitButtonProps {
    hidden?: boolean
    isSubmitting?: boolean
    label?: string
}

export function FormSubmitButton({ hidden, isSubmitting, label }: FormSubmitButtonProps) {
    const { t } = useTranslation('common')

    if (hidden) return null

    return (
        <Button type="submit" disabled={isSubmitting} className="w-full">
            {isSubmitting ? t('actions.saving') : (label ?? t('actions.save'))}
        </Button>
    )
}
