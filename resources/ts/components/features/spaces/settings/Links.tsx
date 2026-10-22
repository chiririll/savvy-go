import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link2, Unlink } from 'lucide-react'
import { SettingsSection } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useLinkSpace, useSpaceLinks, useSpaces, useUnlinkSpace } from '@/hooks'
import { Space } from '@/types/spaces'
import { Confirm } from '@/components/features/spaces/settings/Confirm'

export function Links({ space, isAdmin }: { space: Space; isAdmin: boolean }) {
    const { t } = useTranslation('settings')
    const { data: links = [] } = useSpaceLinks(space.id)
    const { data: mine = [] } = useSpaces()
    const link = useLinkSpace(space.id)
    const unlink = useUnlinkSpace(space.id)
    const [target, setTarget] = useState('')
    const candidates = mine.filter((s) => s.role === 'admin' && s.id !== space.id && !links.some((l) => l.id === s.id))

    return (
        <SettingsSection icon={Link2} title={t('spaces.links.title')} description={t('spaces.links.description')}>
            <div className="space-y-4">
                {links.length ? (
                    <ul className="divide-y rounded-lg border">
                        {links.map((l) => (
                            <li key={l.id} className="flex items-center justify-between gap-3 px-3 py-2 text-sm">
                                <span className="truncate">{l.name}</span>
                                {isAdmin && (
                                    <Confirm
                                        trigger={
                                            <Button variant="ghost" size="sm">
                                                <Unlink className="mr-2 size-4" />
                                                {t('spaces.links.unlink')}
                                            </Button>
                                        }
                                        title={t('spaces.links.unlinkTitle', { name: l.name })}
                                        description={t('spaces.links.unlinkDescription')}
                                        action={t('spaces.links.unlink')}
                                        onConfirm={() => unlink.mutate(l.id)}
                                    />
                                )}
                            </li>
                        ))}
                    </ul>
                ) : (
                    <p className="text-sm text-muted-foreground">{t('spaces.links.none')}</p>
                )}
                {isAdmin &&
                    (candidates.length ? (
                        <div className="flex gap-2">
                            <Select value={target} onValueChange={setTarget}>
                                <SelectTrigger className="flex-1" aria-label={t('spaces.links.choose')}>
                                    <SelectValue placeholder={t('spaces.links.choose')} />
                                </SelectTrigger>
                                <SelectContent>
                                    {candidates.map((c) => (
                                        <SelectItem key={c.id} value={String(c.id)}>
                                            {c.name}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                            <Button
                                disabled={!target || link.isPending}
                                onClick={() => link.mutate(Number(target), { onSuccess: () => setTarget('') })}
                            >
                                {t('spaces.links.link')}
                            </Button>
                        </div>
                    ) : (
                        <p className="text-xs text-muted-foreground">{t('spaces.links.howTo')}</p>
                    ))}
            </div>
        </SettingsSection>
    )
}
