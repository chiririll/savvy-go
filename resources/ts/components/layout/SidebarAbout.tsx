import { Github, ExternalLink } from 'lucide-react'
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem, SidebarFooter } from '@/components/ui/sidebar'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { APP_VERSION } from '@/version'
import { useTranslation } from 'react-i18next'

export function SidebarAbout() {
    const { t } = useTranslation('nav')
    return (
        <SidebarFooter>
            <SidebarMenu>
                <SidebarMenuItem>
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <SidebarMenuButton size="sm" className="text-xs text-muted-foreground justify-between">
                                <span>Savvy</span>
                                <span className="font-mono">{APP_VERSION}</span>
                            </SidebarMenuButton>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent side="top" align="start" className="w-64">
                            <DropdownMenuLabel>{t('about.title')}</DropdownMenuLabel>
                            <DropdownMenuSeparator />
                            <div className="px-2 py-2 text-sm text-muted-foreground">
                                <p>{t('about.body')}</p>
                            </div>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem asChild>
                                <a
                                    href="https://github.com/chiririll/savvy-go"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="cursor-pointer"
                                >
                                    <Github className="mr-2 size-4" />
                                    {t('about.github')}
                                    <ExternalLink className="ml-auto size-3" />
                                </a>
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </SidebarMenuItem>
            </SidebarMenu>
        </SidebarFooter>
    )
}
