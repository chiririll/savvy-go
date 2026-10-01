import { RefreshCw, Star } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { DropdownMenuItem } from '@/components/ui/dropdown-menu'
import { FeedRow, FeedStatusBadge, RowActions } from '@/components/shared'
import { Currency } from '@/types'

const ICON_TONES = [
    'bg-green-100 text-green-600 dark:bg-green-950/40',
    'bg-blue-100 text-blue-600 dark:bg-blue-950/40',
    'bg-orange-100 text-orange-600 dark:bg-orange-950/40',
    'bg-purple-100 text-purple-600 dark:bg-purple-950/40',
    'bg-red-100 text-red-600 dark:bg-red-950/40',
    'bg-sky-100 text-sky-600 dark:bg-sky-950/40',
    'bg-amber-100 text-amber-600 dark:bg-amber-950/40',
    'bg-teal-100 text-teal-600 dark:bg-teal-950/40',
    'bg-rose-100 text-rose-600 dark:bg-rose-950/40',
    'bg-indigo-100 text-indigo-600 dark:bg-indigo-950/40',
] as const

const KNOWN_TONES: Record<string, (typeof ICON_TONES)[number]> = {
    USD: ICON_TONES[0],
    EUR: ICON_TONES[1],
    GBP: ICON_TONES[9],
    JPY: ICON_TONES[4],
    CNY: ICON_TONES[4],
    RUB: ICON_TONES[5],
    CHF: ICON_TONES[4],
    CAD: ICON_TONES[4],
    AUD: ICON_TONES[6],
    NZD: ICON_TONES[6],
    KRW: ICON_TONES[1],
    INR: ICON_TONES[6],
    BRL: ICON_TONES[0],
    TRY: ICON_TONES[4],
    KZT: ICON_TONES[5],
    UAH: ICON_TONES[6],
    PLN: ICON_TONES[8],
    CZK: ICON_TONES[1],
    SEK: ICON_TONES[5],
    NOK: ICON_TONES[1],
    DKK: ICON_TONES[4],
    BTC: ICON_TONES[2],
    ETH: ICON_TONES[3],
    USDT: ICON_TONES[7],
}

function currencyIconTone(code: string): string {
    const known = KNOWN_TONES[code.toUpperCase()]
    if (known) {
        return known
    }

    const hash = [...code.toUpperCase()].reduce((sum, char) => sum + char.charCodeAt(0), 0)
    return ICON_TONES[hash % ICON_TONES.length]
}

interface CurrencyRowProps {
    currency: Currency
    onEdit?: (currency: Currency) => void
    onDelete?: (id: number) => void
    onSetBase?: (id: number) => void
    isSettingBase?: boolean
    isReadOnly?: boolean
    autoUpdated?: boolean
    deleteDisabled?: boolean
    deleteDisabledLabel?: string
}

export function CurrencyRow({
    currency,
    onEdit,
    onDelete,
    onSetBase,
    isSettingBase,
    isReadOnly,
    autoUpdated,
    deleteDisabled,
    deleteDisabledLabel,
}: CurrencyRowProps) {
    const { t } = useTranslation(['common', 'pages'])
    const canEdit = !!onEdit && !isReadOnly
    const canDelete = !!onDelete && !isReadOnly
    const canSetBase = !!onSetBase && !isReadOnly && !currency.isBase
    const rate = currency.isBase ? 1 : currency.rate

    return (
        <FeedRow
            icon={<span className="text-sm font-semibold leading-none">{currency.symbol}</span>}
            iconClassName={currencyIconTone(currency.code)}
            title={currency.name}
            badge={currency.isBase ? (
                <Star
                    className="size-3.5 shrink-0 fill-amber-400 text-amber-500"
                    aria-label={t('pages:currencies.columns.base')}
                />
            ) : undefined}
            meta={(
                <>
                    <FeedStatusBadge variant="secondary">{currency.code}</FeedStatusBadge>
                    {autoUpdated && !currency.isBase && (
                        <span title={t('pages:currencies.autoUpdated')} className="inline-flex">
                            <FeedStatusBadge variant="outline">
                                <RefreshCw className="mr-1 size-3" />
                                {t('pages:currencies.autoShort')}
                            </FeedStatusBadge>
                        </span>
                    )}
                </>
            )}
            amount={rate.toFixed(6)}
            amountClassName={currency.isBase ? 'text-muted-foreground' : undefined}
            onOpen={canEdit ? () => onEdit(currency) : undefined}
            hasActions={canEdit || canDelete || canSetBase}
            actions={({ menuOpen, setMenuOpen, isMobile }) => (
                <RowActions
                    open={menuOpen}
                    onOpenChange={setMenuOpen}
                    showTrigger={!isMobile}
                    onEdit={canEdit ? () => onEdit(currency) : undefined}
                    onDelete={canDelete ? () => onDelete(currency.id) : undefined}
                    deleteTitle={t('pages:currencies.deleteTitle')}
                    deleteDescription={t('pages:currencies.deleteDescription', { name: currency.name })}
                    deleteDisabled={deleteDisabled}
                    deleteDisabledLabel={deleteDisabledLabel}
                >
                    {canSetBase && (
                        <DropdownMenuItem
                            disabled={isSettingBase}
                            onClick={() => onSetBase(currency.id)}
                        >
                            <Star className="mr-2 size-4" />
                            {t('pages:currencies.setAsBase')}
                        </DropdownMenuItem>
                    )}
                </RowActions>
            )}
        />
    )
}
