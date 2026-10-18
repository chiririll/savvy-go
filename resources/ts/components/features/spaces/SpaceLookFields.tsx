import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { EmojiPicker } from '@/components/ui/emoji-picker'
import { Label } from '@/components/ui/label'
import { COLOR_OPTIONS } from '@/constants'
import { cn } from '@/lib/utils'

interface Props {
    /** '' means none. */
    emoji: string
    color: string
    onChange: (look: { emoji: string; color: string }) => void
    disabled?: boolean
}

/** Picks the emoji and color of a space; both may be left empty. */
export function SpaceLookFields({ emoji, color, onChange, disabled }: Props) {
    const { t } = useTranslation('settings')
    return (
        <div className="space-y-3">
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
            <div className="space-y-1.5">
                <Label>{t('spaces.fields.color')}</Label>
                <div className="flex flex-wrap items-center gap-2">
                    {COLOR_OPTIONS.map((c) => (
                        <button
                            key={c}
                            type="button"
                            disabled={disabled}
                            className={cn(
                                'size-7 rounded-full border-2 transition-transform',
                                color === c ? 'scale-110 border-foreground' : 'border-transparent',
                            )}
                            style={{ backgroundColor: c }}
                            onClick={() => onChange({ emoji, color: c })}
                            aria-label={c}
                        />
                    ))}
                    {color && !disabled && (
                        <Button type="button" variant="ghost" size="sm" onClick={() => onChange({ emoji, color: '' })}>
                            {t('spaces.look.clearColor')}
                        </Button>
                    )}
                </div>
            </div>
        </div>
    )
}
