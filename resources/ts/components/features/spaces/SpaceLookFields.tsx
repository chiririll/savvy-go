import { useTranslation } from 'react-i18next'
import { Ban, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { EmojiPicker } from '@/components/shared/pickers/EmojiPicker'
import { Label } from '@/components/ui/label'
import { SpaceBadge } from './SpaceBadge'
import { COLOR_OPTIONS } from '@/constants'
import { cn } from '@/lib/utils'

interface Props {
    /** '' means none. */
    emoji: string
    color: string
    onChange: (look: { emoji: string; color: string }) => void
    disabled?: boolean
    /** The space name, for the letter shown in the preview when there is no emoji. */
    name?: string
}

/** Picks the emoji and color of a space, with a live preview; both may be left empty. */
export function SpaceLookFields({ emoji, color, onChange, disabled, name = '' }: Props) {
    const { t } = useTranslation('settings')
    return (
        <div className="grid gap-4 sm:grid-cols-[auto_1fr] sm:gap-8">
            <div className="flex items-end gap-3 sm:flex-col sm:items-start sm:gap-4">
                <SpaceBadge
                    space={{ name, emoji: emoji || null, color: color || null }}
                    className="size-12 text-xl"
                />
                <span className="sr-only">{t('spaces.look.preview')}</span>
                <div className="space-y-1.5">
                    <Label>{t('spaces.fields.emoji')}</Label>
                    <div className="flex items-center gap-2">
                        <EmojiPicker value={emoji} onChange={(e) => onChange({ emoji: e, color })} disabled={disabled} />
                        {emoji && !disabled && (
                            <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                aria-label={t('spaces.look.clearEmoji')}
                                onClick={() => onChange({ emoji: '', color })}
                            >
                                <X className="size-4" />
                            </Button>
                        )}
                    </div>
                </div>
            </div>
            <div className="space-y-1.5">
                <Label>{t('spaces.fields.color')}</Label>
                <div className="flex flex-wrap items-center gap-2">
                    <button
                        type="button"
                        disabled={disabled}
                        aria-label={t('spaces.look.clearColor')}
                        aria-pressed={!color}
                        className={cn(
                            'flex size-7 items-center justify-center rounded-full border-2 text-muted-foreground transition-transform',
                            !color ? 'scale-110 border-foreground' : 'border-border',
                        )}
                        onClick={() => onChange({ emoji, color: '' })}
                    >
                        <Ban className="size-4" aria-hidden />
                    </button>
                    {COLOR_OPTIONS.map((c) => (
                        <button
                            key={c}
                            type="button"
                            disabled={disabled}
                            aria-pressed={color === c}
                            className={cn(
                                'size-7 rounded-full border-2 transition-transform',
                                color === c ? 'scale-110 border-foreground' : 'border-transparent',
                            )}
                            style={{ backgroundColor: c }}
                            onClick={() => onChange({ emoji, color: c })}
                            aria-label={c}
                        />
                    ))}
                </div>
            </div>
        </div>
    )
}
