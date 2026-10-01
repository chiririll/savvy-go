import { useEffect } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { Home, FolderTree, Coins, CreditCard, Settings, ChevronDown, Receipt, PiggyBank, Hash, BarChart3, HandCoins, Users, Cog, Repeat, Zap, Shield, Upload, Database, LucideIcon, Github, ExternalLink, KeyRound, Activity } from 'lucide-react'
import { Logo } from '@/components/shared/Logo'
import {
    Sidebar,
    SidebarContent,
    SidebarGroup,
    SidebarGroupContent,
    SidebarGroupLabel,
    SidebarHeader,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    SidebarMenuSub,
    SidebarMenuSubItem,
    SidebarMenuSubButton,
    SidebarFooter,
    SidebarRail,
    useSidebar,
} from '@/components/ui/sidebar'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { useUiStore } from '@/stores/ui'
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

interface MenuItem {
    to: string
    icon: LucideIcon
    labelKey: string
}

const mainItems: MenuItem[] = [
    { to: '/', icon: Home, labelKey: 'dashboard' },
    { to: '/transactions', icon: Receipt, labelKey: 'transactions' },
    { to: '/recurring', icon: Repeat, labelKey: 'recurring' },
    { to: '/automation', icon: Zap, labelKey: 'automation' },
    { to: '/budgets', icon: PiggyBank, labelKey: 'budgets' },
    { to: '/debts', icon: HandCoins, labelKey: 'debts' },
    { to: '/reports', icon: BarChart3, labelKey: 'reports' },
]

const settingsItems: MenuItem[] = [
    { to: '/settings/system', icon: Cog, labelKey: 'system' },
    { to: '/settings/monitoring', icon: Activity, labelKey: 'monitoring' },
    { to: '/settings/security', icon: Shield, labelKey: 'security' },
    { to: '/settings/providers', icon: KeyRound, labelKey: 'ssoProviders' },
    { to: '/settings/import', icon: Upload, labelKey: 'import' },
    { to: '/settings/backups', icon: Database, labelKey: 'backups' },
    { to: '/accounts', icon: CreditCard, labelKey: 'accounts' },
    { to: '/categories', icon: FolderTree, labelKey: 'categories' },
    { to: '/currencies', icon: Coins, labelKey: 'currencies' },
    { to: '/tags', icon: Hash, labelKey: 'tags' },
    { to: '/users', icon: Users, labelKey: 'users' },
]

export function AppSidebar() {
    const { t } = useTranslation('nav')
    const location = useLocation()
    const settingsOpen = useUiStore((state) => state.settingsOpen)
    const setSettingsOpen = useUiStore((state) => state.setSettingsOpen)
    const { setOpenMobile } = useSidebar()

    // Close mobile sidebar when navigating to a new page
    useEffect(() => {
        setOpenMobile(false)
    }, [location.pathname, setOpenMobile])

    const isActive = (path: string) => {
        if (path === '/') return location.pathname === '/'
        return location.pathname.startsWith(path)
    }

    return (
        <Sidebar collapsible="icon">
            <SidebarHeader>
                <SidebarMenu>
                    <SidebarMenuItem>
                        <SidebarMenuButton size="lg" asChild>
                            <NavLink to="/">
                                <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                                    <Logo className="size-5" />
                                </div>
                                <div className="grid flex-1 text-left text-sm leading-tight">
                                    <span className="truncate font-semibold">{t('appName', { ns: 'common' })}</span>
                                    <span className="truncate text-xs text-muted-foreground">{t('appTagline', { ns: 'common' })}</span>
                                </div>
                            </NavLink>
                        </SidebarMenuButton>
                    </SidebarMenuItem>
                </SidebarMenu>
            </SidebarHeader>

            <SidebarContent>
                <SidebarGroup>
                    <SidebarGroupLabel>{t('menu')}</SidebarGroupLabel>
                    <SidebarGroupContent>
                        <SidebarMenu>
                            {mainItems.map(({ to, icon: Icon, labelKey }) => {
                                const label = t(labelKey)
                                return (
                                <SidebarMenuItem key={to}>
                                    <SidebarMenuButton
                                        asChild
                                        isActive={isActive(to)}
                                        tooltip={label}
                                    >
                                        <NavLink to={to}>
                                            <Icon />
                                            <span>{label}</span>
                                        </NavLink>
                                    </SidebarMenuButton>
                                </SidebarMenuItem>
                                )
                            })}
                        </SidebarMenu>
                    </SidebarGroupContent>
                </SidebarGroup>

                <SidebarGroup>
                    <SidebarGroupContent>
                        <SidebarMenu>
                            <Collapsible open={settingsOpen} onOpenChange={setSettingsOpen} className="group/collapsible">
                                <SidebarMenuItem>
                                    <CollapsibleTrigger asChild>
                                        <SidebarMenuButton tooltip={t('settings')}>
                                            <Settings />
                                            <span>{t('settings')}</span>
                                            <ChevronDown className={`ml-auto transition-transform duration-200 ${settingsOpen ? '' : '-rotate-90'}`} />
                                        </SidebarMenuButton>
                                    </CollapsibleTrigger>
                                    <CollapsibleContent>
                                        <SidebarMenuSub>
                                            {settingsItems.map(({ to, icon: Icon, labelKey }) => (
                                                <SidebarMenuSubItem key={to}>
                                                    <SidebarMenuSubButton asChild isActive={isActive(to)}>
                                                        <NavLink to={to}>
                                                            <Icon />
                                                            <span>{t(labelKey)}</span>
                                                        </NavLink>
                                                    </SidebarMenuSubButton>
                                                </SidebarMenuSubItem>
                                            ))}
                                        </SidebarMenuSub>
                                    </CollapsibleContent>
                                </SidebarMenuItem>
                            </Collapsible>
                        </SidebarMenu>
                    </SidebarGroupContent>
                </SidebarGroup>
            </SidebarContent>

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
                                        href="https://github.com/chiririll/savvy-plus"
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

            <SidebarRail />
        </Sidebar>
    )
}
