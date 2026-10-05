import { testId, testIdMenu } from '@/lib/test-id'
import type { ReactNode } from 'react'
import { MoreHorizontal, Pencil, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
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

interface RowActionsProps {
    onEdit?: () => void
    onDelete?: () => void
    deleteTitle?: string
    deleteDescription?: string
    deleteDisabled?: boolean
    deleteDisabledLabel?: string
    isReadOnly?: boolean
    leading?: ReactNode
    children?: ReactNode
    open?: boolean
    onOpenChange?: (open: boolean) => void
    showTrigger?: boolean
    triggerClassName?: string
}

export function RowActions({
    onEdit,
    onDelete,
    deleteTitle,
    deleteDescription,
    deleteDisabled,
    deleteDisabledLabel,
    isReadOnly,
    leading,
    children,
    open,
    onOpenChange,
    showTrigger = true,
    triggerClassName,
}: RowActionsProps) {
    const { t } = useTranslation()
    const canDelete = Boolean(onDelete) && !isReadOnly

    return (
        <DropdownMenu open={open} onOpenChange={onOpenChange}>
            <DropdownMenuTrigger asChild>
                <Button
                    variant="ghost"
                    size="icon"
                    {...testIdMenu('row')}
                    className={cn(
                        'size-8',
                        !showTrigger && 'pointer-events-none absolute right-1 top-1/2 -translate-y-1/2 opacity-0',
                        triggerClassName,
                    )}
                    tabIndex={showTrigger ? undefined : -1}
                    aria-hidden={!showTrigger}
                >
                    <MoreHorizontal className="size-4" />
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
                {leading}
                {onEdit && (
                    <DropdownMenuItem onClick={onEdit} {...testId('edit')}>
                        <Pencil className="mr-2 size-4" />
                        {t('actions.edit')}
                    </DropdownMenuItem>
                )}
                {children}
                {canDelete && (
                    <>
                        {(onEdit || leading || children) && <DropdownMenuSeparator />}
                        <AlertDialog>
                            <AlertDialogTrigger asChild>
                                <DropdownMenuItem
                                    variant="destructive"
                                    {...testId('delete')}
                                    onSelect={(event) => event.preventDefault()}
                                    disabled={deleteDisabled}
                                >
                                    <Trash2 className="mr-2 size-4" />
                                    {deleteDisabled
                                        ? (deleteDisabledLabel ?? t('actions.delete'))
                                        : t('actions.delete')}
                                </DropdownMenuItem>
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                                <AlertDialogHeader>
                                    <AlertDialogTitle>{deleteTitle ?? t('actions.delete')}</AlertDialogTitle>
                                    {deleteDescription && (
                                        <AlertDialogDescription>{deleteDescription}</AlertDialogDescription>
                                    )}
                                </AlertDialogHeader>
                                <AlertDialogFooter>
                                    <AlertDialogCancel>{t('actions.cancel')}</AlertDialogCancel>
                                    <AlertDialogAction variant="destructive" onClick={onDelete}>
                                        {t('actions.delete')}
                                    </AlertDialogAction>
                                </AlertDialogFooter>
                            </AlertDialogContent>
                        </AlertDialog>
                    </>
                )}
            </DropdownMenuContent>
        </DropdownMenu>
    )
}
