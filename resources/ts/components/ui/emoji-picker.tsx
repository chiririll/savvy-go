import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import EmojiPickerReact, { EmojiClickData, EmojiStyle, Theme } from 'emoji-picker-react'
import { Popover, PopoverContent, PopoverTrigger } from './popover'
import { Button } from './button'

interface EmojiPickerProps {
    value: string
    onChange: (emoji: string) => void
    disabled?: boolean
}

export function EmojiPicker({ value, onChange, disabled }: EmojiPickerProps) {
    const { t } = useTranslation()
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
                    className="size-10 text-xl"
                >
                    {value || '😀'}
                </Button>
            </PopoverTrigger>
            <PopoverContent className="w-auto p-0" align="start">
                <EmojiPickerReact
                    onEmojiClick={handleEmojiClick}
                    theme={Theme.LIGHT}
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
