import { LocateFixed } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

interface UseBrowserUrlButtonProps {
    /** The address to offer, normally `window.location.origin`. */
    url: string
    /** The URL currently in the field. The button hides once it already equals `url`. */
    current: string
    onUse: (url: string) => void
}

/**
 * Icon button for the inside of a URL input: place it in a `relative` wrapper and give the input right padding.
 * Offers the address the admin opened the app at. Only fills the field; saving stays explicit.
 */
export function UseBrowserUrlButton({ url, current, onUse }: UseBrowserUrlButtonProps) {
    const { t } = useTranslation('settings')
    if (current.trim() === url) return null
    const label = t('system.server.useBrowserUrl', { url })

    return (
        <Tooltip>
            <TooltipTrigger asChild>
                <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    aria-label={label}
                    className="absolute top-1/2 right-1 size-7 -translate-y-1/2 text-muted-foreground"
                    onClick={() => onUse(url)}
                >
                    <LocateFixed className="size-4" />
                </Button>
            </TooltipTrigger>
            <TooltipContent>{label}</TooltipContent>
        </Tooltip>
    )
}
