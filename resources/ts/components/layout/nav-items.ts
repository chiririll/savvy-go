import {
    Home,
    FolderTree,
    Coins,
    CreditCard,
    Receipt,
    PiggyBank,
    Hash,
    BarChart3,
    HandCoins,
    Users,
    Cog,
    Repeat,
    Zap,
    Shield,
    Upload,
    Database,
    LucideIcon,
    Activity,
    KeyRound,
    UserRound,
    Boxes,
    ServerCog,
    HardDrive,
    ClipboardCheck,
} from 'lucide-react'
import { type NavKey } from '@/lib/labels'

export interface MenuItem {
    to: string
    icon: LucideIcon
    labelKey: NavKey
    /** Other paths that belong to this item (sub-pages). */
    also?: string[]
}

export const mainItems: MenuItem[] = [
    { to: '/', icon: Home, labelKey: 'dashboard' },
    { to: '/transactions', icon: Receipt, labelKey: 'transactions' },
    { to: '/recurring', icon: Repeat, labelKey: 'recurring' },
    { to: '/budgets', icon: PiggyBank, labelKey: 'budgets' },
    { to: '/debts', icon: HandCoins, labelKey: 'debts' },
    { to: '/reports', icon: BarChart3, labelKey: 'reports' },
]

/** The current space's setup, and the user's own settings. */
export const settingsItems: MenuItem[] = [
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
export const adminItems: MenuItem[] = [
    { to: '/admin/checks', icon: ClipboardCheck, labelKey: 'checks' },
    { to: '/admin/system', icon: Cog, labelKey: 'system' },
    { to: '/admin/monitoring', icon: Activity, labelKey: 'monitoring' },
    { to: '/admin/sso', icon: KeyRound, labelKey: 'sso', also: ['/admin/providers'] },
    { to: '/admin/security', icon: Shield, labelKey: 'security' },
    { to: '/admin/backups', icon: HardDrive, labelKey: 'systemBackups' },
    { to: '/admin/users', icon: Users, labelKey: 'users' },
    { to: '/admin/spaces', icon: ServerCog, labelKey: 'spaces' },
]
