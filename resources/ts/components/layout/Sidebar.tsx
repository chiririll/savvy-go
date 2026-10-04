import { useEffect } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { Home, FolderTree, Coins, CreditCard, Settings, ChevronDown, Receipt, PiggyBank, Hash, BarChart3, HandCoins, Users, Cog, Repeat, Zap, Shield, Upload, Database, LucideIcon, Github, ExternalLink, Activity, KeyRound, UserRound, Boxes, ServerCog, HardDrive, Wrench } from 'lucide-react'
import { SpaceSwitcher } from '@/components/features/spaces/SpaceSwitcher'
import { useUser } from '@/stores/auth'
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
    /** Other paths that belong to this item (sub-pages). */
    also?: string[]
}

const mainItems: MenuItem[] = [
    { to: '/', icon: Home, labelKey: 'dashboard' },
    { to: '/transactions', icon: Receipt, labelKey: 'transactions' },
    { to: '/recurring', icon: Repeat, labelKey: 'recurring' },
    { to: '/budgets', icon: PiggyBank, labelKey: 'budgets' },
    { to: '/debts', icon: HandCoins, labelKey: 'debts' },
    { to: '/reports', icon: BarChart3, labelKey: 'reports' },
]

/** The current space's setup, and the user's own settings. */
const settingsItems: MenuItem[] = [
    { to: '/accounts', icon: CreditCard, labelKey: 'accounts' },
    { to: '/categories', icon: FolderTree, labelKey: 'categories' },
    { to: '/tags', icon: Hash, labelKey: 'tags' },
    { to: '/currencies', icon: Coins, labelKey: 'currencies' },
    { to: '/automation', icon: Zap, labelKey: 'automation' },
    { to: '/settings/space', icon: Boxes, labelKey: 'space' },
    { to: '/settings/user', icon: UserRound, labelKey: 'user', also: ['/settings/security', '/settings/api'] },
    { to: '/settings/import', icon: Upload, labelKey: 'import' },
    { to: '/settings/backups', icon: Database, labelKey: 'backups' },
]

/** The server: server administrators only. */
const adminItems: MenuItem[] = [
    { to: '/admin/system', icon: Cog, labelKey: 'system' },
    { to: '/admin/monitoring', icon: Activity, labelKey: 'monitoring' },
    { to: '/admin/sso', icon: KeyRound, labelKey: 'sso', also: ['/admin/providers'] },
    { to: '/admin/security', icon: Shield, labelKey: 'security' },
    { to: '/admin/backups', icon: HardDrive, labelKey: 'systemBackups' },
    { to: '/admin/users', icon: Users, labelKey: 'users' },
    { to: '/admin/spaces', icon: ServerCog, labelKey: 'spaces' },
]

export function AppSidebar() {
    const { t } = useTranslation('nav')
    const location = useLocation()
    const settingsOpen = useUiStore((state) => state.settingsOpen)
    const setSettingsOpen = useUiStore((state) => state.setSettingsOpen)
    const adminOpen = useUiStore((state) => state.adminOpen)
    const setAdminOpen = useUiStore((state) => state.setAdminOpen)
    const isServerAdmin = useUser()?.role === 'admin'
    const { setOpenMobile } = useSidebar()

    // Close mobile sidebar when navigating to a new page
    useEffect(() => {
        setOpenMobile(false)
    }, [location.pathname, setOpenMobile])

    const isActive = (path: string, also: string[] = []) => {
        if (path === '/') return location.pathname === '/'
        return [path, ...also].some((p) => location.pathname.startsWith(p))
    }

    return (
        <Sidebar collapsible="icon">
            <SidebarHeader>
                <SpaceSwitcher />
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

                <NavGroup
                    icon={Settings}
                    label={t('settings')}
                    items={settingsItems}
                    open={settingsOpen}
                    onOpenChange={setSettingsOpen}
                    isActive={isActive}
                />
                {isServerAdmin && (
                    <NavGroup
                        icon={Wrench}
                        label={t('administration')}
                        items={adminItems}
                        open={adminOpen}
                        onOpenChange={setAdminOpen}
                        isActive={isActive}
                    />
                )}
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

            <SidebarRail />
        </Sidebar>
    )
}

function NavGroup({
    icon: GroupIcon,
    label,
    items,
    open,
    onOpenChange,
    isActive,
}: {
    icon: LucideIcon
    label: string
    items: MenuItem[]
    open: boolean
    onOpenChange: (open: boolean) => void
    isActive: (path: string, also?: string[]) => boolean
}) {
    const { t } = useTranslation('nav')
    return (
        <SidebarGroup>
            <SidebarGroupContent>
                <SidebarMenu>
                    <Collapsible open={open} onOpenChange={onOpenChange} className="group/collapsible">
                        <SidebarMenuItem>
                            <CollapsibleTrigger asChild>
                                <SidebarMenuButton tooltip={label}>
                                    <GroupIcon />
                                    <span>{label}</span>
                                    <ChevronDown className={`ml-auto transition-transform duration-200 ${open ? '' : '-rotate-90'}`} />
                                </SidebarMenuButton>
                            </CollapsibleTrigger>
                            <CollapsibleContent>
                                <SidebarMenuSub>
                                    {items.map(({ to, icon: Icon, labelKey, also }) => (
                                        <SidebarMenuSubItem key={to}>
                                            <SidebarMenuSubButton asChild isActive={isActive(to, also)}>
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
    )
}
