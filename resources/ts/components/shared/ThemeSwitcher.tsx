import { Moon, Sun } from 'lucide-react'
import { useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuRadioGroup,
    DropdownMenuRadioItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useTheme, type Theme, type ThemePreference } from '@/hooks/use-theme'
import { cn } from '@/lib/utils'

function ThemeGlyph({ theme, className }: { theme: Theme; className?: string }) {
    const Icon = theme === 'dark' ? Moon : Sun
    return <Icon className={className} />
}

/** Regular sun/moon glyph with an "A" badge so Auto is readable in both themes. */
function AutoThemeIcon({
    theme,
    className,
    badgeClassName,
}: {
    theme: Theme
    className?: string
    badgeClassName?: string
}) {
    return (
        <span className={cn('relative inline-flex', className)} aria-hidden>
            <ThemeGlyph theme={theme} className="size-full" />
            <span
                className={cn(
                    'absolute -right-0.5 -bottom-0.5 flex items-center justify-center rounded-[2px] bg-background font-bold leading-none text-foreground ring-1 ring-border',
                    badgeClassName,
                )}
            >
                A
            </span>
        </span>
    )
}

function TriggerIcon({ preference, theme }: { preference: ThemePreference; theme: Theme }) {
    if (preference === 'auto') {
        return <AutoThemeIcon theme={theme} className="size-5" badgeClassName="size-2.5 text-[8px]" />
    }
    return <ThemeGlyph theme={theme} className="h-5 w-5" />
}

interface ThemeSwitcherProps {
    /** `select` is a labelled settings row (like the language one); `icon` is the header button. */
    variant?: 'icon' | 'select'
    className?: string
}

export function ThemeSwitcher({ variant = 'icon', className }: ThemeSwitcherProps) {
    const { t } = useTranslation()
    const { theme, preference, setTheme } = useTheme()
    const triggerRef = useRef<HTMLButtonElement>(null)

    const handleChange = (value: string) => {
        if (value !== 'light' && value !== 'dark' && value !== 'auto') return
        // The select row has no trigger button to ripple from: fade instead.
        const rect = variant === 'icon' ? triggerRef.current?.getBoundingClientRect() : undefined
        setTheme(value, rect && { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 })
    }

    if (variant === 'select') {
        return (
            <div className={cn('flex items-center justify-between gap-4 px-4 py-3.5', className)}>
                <div className="space-y-1">
                    <Label htmlFor="interface-theme" className="text-sm font-medium">
                        {t('theme.label')}
                    </Label>
                    <p className="text-sm leading-relaxed text-muted-foreground">{t('settings:user.theme.description')}</p>
                </div>
                <Select value={preference} onValueChange={handleChange}>
                    <SelectTrigger id="interface-theme" className="w-40">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="light">{t('theme.light')}</SelectItem>
                        <SelectItem value="dark">{t('theme.dark')}</SelectItem>
                        <SelectItem value="auto">{t('theme.auto')}</SelectItem>
                    </SelectContent>
                </Select>
            </div>
        )
    }

    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button
                    ref={triggerRef}
                    variant="ghost"
                    size="icon"
                    className={cn('overflow-visible', className)}
                    aria-label={t('theme.label')}
                >
                    <TriggerIcon preference={preference} theme={theme} />
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
                <DropdownMenuRadioGroup value={preference} onValueChange={handleChange}>
                    <DropdownMenuRadioItem value="light">
                        <Sun className="size-4" />
                        {t('theme.light')}
                    </DropdownMenuRadioItem>
                    <DropdownMenuRadioItem value="dark">
                        <Moon className="size-4" />
                        {t('theme.dark')}
                    </DropdownMenuRadioItem>
                    <DropdownMenuRadioItem value="auto">
                        <AutoThemeIcon theme={theme} className="size-4" badgeClassName="size-2.5 text-[7px]" />
                        {t('theme.auto')}
                    </DropdownMenuRadioItem>
                </DropdownMenuRadioGroup>
            </DropdownMenuContent>
        </DropdownMenu>
    )
}
