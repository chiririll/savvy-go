import { Fragment, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Eye, TriangleAlert, X, type LucideIcon } from 'lucide-react'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import { isNonProductionApp } from '@/lib/app-env'

interface Status {
    id: string
    icon: LucideIcon
    label: string
    /** Whether the condition for this status holds right now. */
    shown: boolean
    /** The user may hide it; it stays hidden in this browser. */
    dismissible?: boolean
}

const HIDDEN_KEY = 'savvy-hidden-statuses'

function readHidden(): string[] {
    try {
        const hidden: unknown = JSON.parse(localStorage.getItem(HIDDEN_KEY) ?? '[]')
        return Array.isArray(hidden) ? hidden.filter((id): id is string => typeof id === 'string') : []
    } catch {
        return []
    }
}

/** Everything the bar can report; add a status here to show it. */
function useStatuses(): Status[] {
    const { t } = useTranslation()
    const isReadOnly = useReadOnly()

    return [
        { id: 'devMode', icon: TriangleAlert, label: t('devMode.banner'), shown: isNonProductionApp() },
        { id: 'readOnly', icon: Eye, label: t('readOnly.banner'), shown: isReadOnly, dismissible: true },
    ]
}

/** One bar above the header for every active status, separated by dots. */
export function StatusBar() {
    const { t } = useTranslation()
    const [hidden, setHidden] = useState(readHidden)
    const statuses = useStatuses().filter((status) => status.shown && !hidden.includes(status.id))
    const dismissible = statuses.filter((status) => status.dismissible).map((status) => status.id)

    if (statuses.length === 0) {
        return null
    }

    const dismiss = () => {
        const next = [...hidden, ...dismissible]
        setHidden(next)
        try {
            localStorage.setItem(HIDDEN_KEY, JSON.stringify(next))
        } catch {
            // Hidden for this session only.
        }
    }

    return (
        <div
            role="status"
            className="relative flex w-full min-w-0 items-center justify-center bg-amber-500 px-9 py-1.5 text-center text-sm font-medium text-amber-950"
        >
            <p className="min-w-0">
                {statuses.map(({ id, icon: Icon, label }, index) => (
                    <Fragment key={id}>
                        {index > 0 && <span aria-hidden className="mx-2">·</span>}
                        <Icon className="mr-1.5 inline size-4 -translate-y-px align-middle" />
                        {label}
                    </Fragment>
                ))}
            </p>
            {dismissible.length > 0 && (
                <button
                    type="button"
                    onClick={dismiss}
                    className="absolute right-2 rounded p-1 transition-colors hover:bg-amber-600/20"
                    aria-label={t('readOnly.close')}
                >
                    <X className="size-4" />
                </button>
            )}
        </div>
    )
}
