import { useState } from 'react'
import { SmilePlus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import EmojiPickerReact, { EmojiClickData, EmojiStyle, Theme } from 'emoji-picker-react'
import { Popover, PopoverContent, PopoverTrigger } from '../../ui/popover'
import { Button } from '../../ui/button'
import { useTheme } from '@/hooks/use-theme'

interface EmojiPickerProps {
    value: string
    onChange: (emoji: string) => void
    disabled?: boolean
    /** The id of the element that names the button. */
    labelledBy?: string
}

export function EmojiPicker({ value, onChange, disabled, labelledBy }: EmojiPickerProps) {
    const { t } = useTranslation()
    const { theme } = useTheme()
    const [open, setOpen] = useState(false)

    const handleEmojiClick = (emojiData: EmojiClickData) => {
        onChange(emojiData.emoji)
        setOpen(false)
    }

    return (
        <Popover modal open={open} onOpenChange={setOpen}>
            <PopoverTrigger asChild>
                <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    disabled={disabled}
                    aria-labelledby={labelledBy}
                    className="size-10 text-xl"
                >
                    {value || <SmilePlus data-testid="emoji-placeholder" aria-hidden className="size-5 text-muted-foreground" />}
                </Button>
            </PopoverTrigger>
            <PopoverContent className="w-auto p-0" align="start" aria-label={t('fields.icon')}>
                <EmojiPickerReact
                    onEmojiClick={handleEmojiClick}
                    theme={theme === 'dark' ? Theme.DARK : Theme.LIGHT}
                    emojiStyle={EmojiStyle.NATIVE}
                    lazyLoadEmojis
                    width={350}
                    height={400}
                    searchPlaceHolder={t('searchEmoji')}
                    previewConfig={{ showPreview: false }}
                />
            </PopoverContent>
        </Popover>
    )
}
