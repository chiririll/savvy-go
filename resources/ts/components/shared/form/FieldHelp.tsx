import { HelpCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { cn } from '@/lib/utils'

export function FieldHelp({ children }: { children: React.ReactNode }) {
    const { t } = useTranslation('common')

    return (
        <Popover modal={false}>
            <PopoverTrigger asChild>
                <button
                    type="button"
                    aria-label={t('actions.help')}
                    className={cn(
                        'inline-flex size-4 shrink-0 items-center justify-center text-muted-foreground',
                        'hover:text-foreground',
                    )}
                >
                    <HelpCircle className="size-3.5" />
                </button>
            </PopoverTrigger>
            <PopoverContent
                side="top"
                aria-label={t('actions.help')}
                align="center"
                collisionPadding={8}
                onOpenAutoFocus={(event) => event.preventDefault()}
                onCloseAutoFocus={(event) => event.preventDefault()}
                className={cn(
                    'z-[200] w-fit max-w-[min(18rem,calc(100vw-2rem))] border-none bg-foreground px-3',
                    'py-1.5 text-xs text-balance text-background shadow-md',
                )}
            >
                {children}
            </PopoverContent>
        </Popover>
    )
}
