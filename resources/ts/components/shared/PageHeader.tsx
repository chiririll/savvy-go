import { testId } from '@/lib/test-id'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Link } from 'react-router-dom'
import { Plus, ArrowLeft } from 'lucide-react'
import { Separator } from '@/components/ui/separator'

interface Props {
    title: string
    description?: string
    createLink?: string
    createLabel?: string
    onCreateClick?: () => void
    backLink?: string
    actions?: React.ReactNode
}

export function PageHeader({
                               title,
                               description,
                               createLink,
                               createLabel,
                               onCreateClick,
                               backLink,
                               actions
                           }: Props) {
    const { t } = useTranslation()
    const label = createLabel ?? t('actions.create')
    const createButton = onCreateClick ? (
        <Button onClick={onCreateClick} {...testId('create')}>
            <Plus className="mr-2 h-4 w-4" />
            {label}
        </Button>
    ) : createLink ? (
        <Button asChild>
            <Link to={createLink}>
                <Plus className="mr-2 h-4 w-4" />
                {label}
            </Link>
        </Button>
    ) : null
    return (
        <div className="mb-4 sm:mb-8">
            <div className="flex items-center justify-between gap-3">
                <div className="flex min-w-0 items-center gap-4">
                    {backLink && (
                        <Button variant="ghost" size="icon" asChild>
                            <Link to={backLink}>
                                <ArrowLeft className="h-4 w-4" />
                            </Link>
                        </Button>
                    )}
                    <div className="min-w-0">
                        <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">{title}</h1>
                        {description && (
                            <p className="text-muted-foreground mt-1 hidden sm:block">{description}</p>
                        )}
                    </div>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                    {actions}
                    {createButton}
                </div>
            </div>
            <Separator className="mt-4 sm:mt-6" />
        </div>
    )
}
