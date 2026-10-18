import { NavLink } from 'react-router-dom'
import { ChevronDown, LucideIcon } from 'lucide-react'
import {
    SidebarGroup,
    SidebarGroupContent,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    SidebarMenuSub,
    SidebarMenuSubItem,
    SidebarMenuSubButton,
} from '@/components/ui/sidebar'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { useTranslation } from 'react-i18next'
import { navLabel } from '@/lib/labels'
import { cn } from '@/lib/utils'

import type { MenuItem } from './nav-items'

export function NavGroup({
    icon: GroupIcon,
    label,
    items,
    alert,
    counts,
    open,
    onOpenChange,
    isActive,
}: {
    icon: LucideIcon
    label: string
    items: MenuItem[]
    /** Something in the group needs attention: shows a dot on the header. */
    alert?: 'error' | 'warning'
    /** Number shown next to an item, by its path; zero shows nothing. */
    counts?: Record<string, number>
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
                                    {alert && (
                                        <span
                                            role="img"
                                            aria-label={t('needsAttention')}
                                            className={cn(
                                                'ml-auto size-2 rounded-full',
                                                alert === 'error' ? 'bg-red-500' : 'bg-amber-500',
                                            )}
                                        />
                                    )}
                                    <ChevronDown
                                        className={cn(
                                            'transition-transform duration-200',
                                            !alert && 'ml-auto',
                                            !open && '-rotate-90',
                                        )}
                                    />
                                </SidebarMenuButton>
                            </CollapsibleTrigger>
                            <CollapsibleContent>
                                <SidebarMenuSub>
                                    {items.map(({ to, icon: Icon, labelKey, also }) => (
                                        <SidebarMenuSubItem key={to}>
                                            <SidebarMenuSubButton asChild isActive={isActive(to, also)}>
                                                <NavLink to={to}>
                                                    <Icon />
                                                    <span>{navLabel(t, labelKey)}</span>
                                                    {!!counts?.[to] && (
                                                        <span
                                                            className={cn(
                                                                'ml-auto rounded-full bg-amber-500/15 px-1.5 text-xs',
                                                                'font-medium tabular-nums',
                                                                'text-amber-600 dark:text-amber-400',
                                                            )}
                                                        >
                                                            {counts[to]}
                                                        </span>
                                                    )}
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
