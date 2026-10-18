import { useTranslation } from 'react-i18next'
import { TriangleAlert } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { useSyncGroups } from '@/hooks/use-spaces'
import type { SpaceChoice } from './SpacePicker'

/**
 * Categories and tags of spaces in a sync group are kept identical, so a token
 * that changes them in one space changes the others too. Says which spaces that
 * touches, and which the token cannot reach (there it is refused).
 */
export function SyncWarning({
    spaces,
    choice,
}: {
    spaces: ReadonlyArray<{ id: number; name: string }>
    choice: SpaceChoice
}) {
    const { t } = useTranslation('settings')
    const chosen = spaces.filter((s) => choice.all || choice.ids.includes(s.id))
    const groups = useSyncGroups(chosen.map((s) => s.id))

    const notes: { key: string; unreachable: boolean; text: string }[] = []
    chosen.forEach((space, i) => {
        const group = groups[i]
        if (!group || (!group.categories && !group.tags)) return
        const others = group.members.filter((m) => m.id !== space.id)
        const missing = others.filter((m) => !choice.all && !choice.ids.includes(m.id)).map((m) => m.name)
        const what = group.categories && group.tags ? 'both' : group.categories ? 'categories' : 'tags'
        const kind =
            what === 'both' ? t('api.sync.both') : what === 'categories' ? t('api.sync.categories') : t('api.sync.tags')
        const names = others.map((m) => m.name).join(', ')
        if (missing.length > 0 || group.hidden > 0) {
            notes.push({
                key: `${space.id}-x`,
                unreachable: true,
                text: t('api.sync.unreachable', {
                    space: space.name,
                    kind,
                    others: [
                        ...missing,
                        ...(group.hidden > 0 ? [t('api.sync.hidden', { count: group.hidden })] : []),
                    ].join(', '),
                }),
            })
        } else {
            notes.push({
                key: `${space.id}-ok`,
                unreachable: false,
                text: t('api.sync.mirrored', { space: space.name, kind, others: names }),
            })
        }
    })
    if (notes.length === 0) return null

    return (
        <Alert>
            <TriangleAlert className="size-4" />
            <AlertDescription className="space-y-1">
                <p className="font-medium">{t('api.sync.title')}</p>
                {notes.map((n) => (
                    <p key={n.key} className={n.unreachable ? 'text-destructive' : undefined}>
                        {n.text}
                    </p>
                ))}
            </AlertDescription>
        </Alert>
    )
}
