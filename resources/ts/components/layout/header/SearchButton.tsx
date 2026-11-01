import { lazy, Suspense, useEffect, useState } from 'react'
import { Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { useIsMobile } from '@/hooks/use-mobile'
import { shouldOpenSearchPalette } from '@/lib/search-shortcut'

const TransactionSearchPalette = lazy(() =>
    import('@/components/features/transactions/search/TransactionSearchPalette').then((module) => ({
        default: module.TransactionSearchPalette,
    })),
)

/** Desktop header button, and the `/` shortcut, that open the transaction search palette. */
export function SearchButton() {
    const { t } = useTranslation('nav')
    const isMobile = useIsMobile()
    const [open, setOpen] = useState(false)

    useEffect(() => {
        if (isMobile) return
        const onKeyDown = (event: KeyboardEvent) => {
            const modalOpen = document.querySelector('[role="dialog"], [role="alertdialog"]') !== null
            if (!shouldOpenSearchPalette(event, modalOpen)) return
            // Firefox would start Quick Find.
            event.preventDefault()
            setOpen(true)
        }
        document.addEventListener('keydown', onKeyDown)
        return () => document.removeEventListener('keydown', onKeyDown)
    }, [isMobile])

    if (isMobile) {
        return null
    }

    return (
        <>
            <Button
                variant="outline"
                size="sm"
                className="gap-2 text-muted-foreground"
                aria-label={t('search')}
                aria-keyshortcuts="/"
                onClick={() => setOpen(true)}
            >
                <Search className="size-4" />
                <span className="hidden lg:inline">{t('search')}</span>
                <kbd className="rounded border px-1.5 text-xs">/</kbd>
            </Button>
            {open && (
                <Suspense fallback={null}>
                    <TransactionSearchPalette open={open} onOpenChange={setOpen} />
                </Suspense>
            )}
        </>
    )
}
