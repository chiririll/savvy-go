import { TFunction } from 'i18next'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

/** What a Laravel database is restored as: the whole server, the current space, or a new space. */
export type LaravelTarget = 'server' | 'space' | 'new'

/** True for a bare database file, which in practice is a database of the Laravel version. */
export const isLaravelFile = (file: File | null) => !!file && file.name.toLowerCase().endsWith('.sqlite')

/** Title and explanation of each target. */
const targetTexts = (t: TFunction) => ({
    server: { title: t('backups.laravel.target.server.title'), description: t('backups.laravel.target.server.description') },
    space: { title: t('backups.laravel.target.space.title'), description: t('backups.laravel.target.space.description') },
    new: { title: t('backups.laravel.target.new.title'), description: t('backups.laravel.target.new.description') },
} satisfies Record<LaravelTarget, { title: string; description: string }>)

/** The label of the button that restores as the target. */
export const laravelActionLabel = (t: TFunction, target: LaravelTarget): string => ({
    server: t('backups.laravel.action.server'),
    space: t('backups.laravel.action.space'),
    new: t('backups.laravel.action.new'),
} satisfies Record<LaravelTarget, string>)[target]

interface LaravelRestoreOptionsProps {
    targets: LaravelTarget[]
    target: LaravelTarget
    onTarget: (target: LaravelTarget) => void
    appKey: string
    onAppKey: (value: string) => void
}

/**
 * A Laravel database is not kept as a backup: it is converted and restored at once.
 * This asks what to restore it as and, for the whole server, for the APP_KEY that
 * decrypts the two-factor secrets.
 */
export function LaravelRestoreOptions({ targets, target, onTarget, appKey, onAppKey }: LaravelRestoreOptionsProps) {
    const { t } = useTranslation('settings')
    const texts = targetTexts(t)
    return (
        <div className="space-y-4">
            <p className="text-sm text-muted-foreground">{t('backups.laravel.intro')}</p>
            <div role="radiogroup" className="space-y-2">
                {targets.map((option) => (
                    <label key={option} className="flex cursor-pointer items-start gap-3 rounded-md border p-3 has-[:checked]:border-primary">
                        <input
                            type="radio"
                            name="laravel-target"
                            className="mt-1"
                            checked={target === option}
                            onChange={() => onTarget(option)}
                        />
                        <span className="space-y-1">
                            <span className="block text-sm font-medium">{texts[option].title}</span>
                            <span className="block text-xs text-muted-foreground">{texts[option].description}</span>
                        </span>
                    </label>
                ))}
            </div>
            {target === 'server' && (
                <div className="space-y-2">
                    <Label htmlFor="laravel-app-key">{t('backups.laravel.appKey')}</Label>
                    <Input
                        id="laravel-app-key"
                        type="password"
                        autoComplete="off"
                        placeholder="base64:…"
                        value={appKey}
                        onChange={(e) => onAppKey(e.target.value)}
                    />
                    <p className="text-xs text-muted-foreground">{t('backups.laravel.appKeyHint')}</p>
                </div>
            )}
        </div>
    )
}
