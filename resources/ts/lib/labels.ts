import type { TFunction } from 'i18next'

import type { AccountType } from '@/types/accounts'
import type { ApiTokenScope } from '@/types/api-tokens'
import type { ActionType, ConditionOperator, TriggerType } from '@/types/automation'
import type { BackupStatus } from '@/types/backup'
import type { BudgetPeriod } from '@/types/budgets'
import type { CategoryType } from '@/types/categories'
import type { DebtType } from '@/types/debts'
import type { ImportStep } from '@/types/import'
import type { RecurringFrequency } from '@/types/recurring'
import type { SpaceRole, TransferReview } from '@/types/spaces'
import type { TransactionType } from '@/types/transactions'
import type { UserRole } from '@/types/users'

/*
 * Labels for enum-like values. Every key is spelled out in a literal call, so
 * the locale tests can check it exists; never build a key from a value. A
 * `Record<Union, string>` makes the compiler demand a label for each new value.
 * The functions take the translate function so they work in components and outside them alike.
 */

/** For values the server may send that this client does not know: show them as they are. */
const orRaw = (labels: Record<string, string>, value: string): string => labels[value] ?? value

export type PeriodType = 'last_30_days' | 'month' | 'quarter' | 'year' | 'ytd' | 'custom'
export type GroupBy = 'day' | 'week' | 'month'
export type ReportTab = 'overview' | 'cashflow' | 'expenses' | 'income' | 'networth'
export type SortOption = 'dateNewest' | 'dateOldest' | 'amountHigh' | 'amountLow'
export type UserTab = 'preferences' | 'security' | 'api'
export type AutomationLogStatus = 'success' | 'error' | 'skipped'
export type SsoPresetKey = 'custom_oidc' | 'custom_saml'
export type ConditionField = 'type' | 'amount' | 'description' | 'account_id' | 'category_id' | 'tags'

export const SSO_ERROR_CODES = [
    'audience_mismatch',
    'discovery_failed',
    'email_in_use',
    'email_unverified',
    'id_token_invalid',
    'id_token_missing',
    'idp_error',
    'invalid_request',
    'invalid_state',
    'issuer_mismatch',
    'jwks_failed',
    'jwks_missing',
    'missing_field',
    'missing_subject',
    'no_admin',
    'no_email',
    'saml_invalid',
    'signup_disabled',
    'sso_error',
    'token_exchange_failed',
    'user_missing',
    'userinfo_failed',
] as const
export type SsoErrorCode = (typeof SSO_ERROR_CODES)[number]

export const accountTypeLabel = (t: TFunction, type: AccountType): string => ({
    bank: t('pages:accounts.types.bank'),
    cash: t('pages:accounts.types.cash'),
    crypto: t('pages:accounts.types.crypto'),
    debt: t('pages:accounts.types.debt'),
} satisfies Record<AccountType, string>)[type]

/** Account types come from the server, so an unknown one falls back to its raw value. */
export const accountTypeLabelLoose = (t: TFunction, type: string): string =>
    orRaw({
        bank: t('pages:accounts.types.bank'),
        cash: t('pages:accounts.types.cash'),
        crypto: t('pages:accounts.types.crypto'),
        debt: t('pages:accounts.types.debt'),
    }, type)

export const apiScopeLabel = (t: TFunction, scope: ApiTokenScope): string => ({
    'read': t('settings:api.scope.read'),
    'read-write': t('settings:api.scope.read-write'),
} satisfies Record<ApiTokenScope, string>)[scope]

export const apiScopeHint = (t: TFunction, scope: ApiTokenScope): string => ({
    'read': t('settings:api.create.accessHint.read'),
    'read-write': t('settings:api.create.accessHint.read-write'),
} satisfies Record<ApiTokenScope, string>)[scope]

export const ssoErrorLabel = (t: TFunction, code: SsoErrorCode): string => ({
    audience_mismatch: t('auth:sso.errors.audience_mismatch'),
    discovery_failed: t('auth:sso.errors.discovery_failed'),
    email_in_use: t('auth:sso.errors.email_in_use'),
    email_unverified: t('auth:sso.errors.email_unverified'),
    id_token_invalid: t('auth:sso.errors.id_token_invalid'),
    id_token_missing: t('auth:sso.errors.id_token_missing'),
    idp_error: t('auth:sso.errors.idp_error'),
    invalid_request: t('auth:sso.errors.invalid_request'),
    invalid_state: t('auth:sso.errors.invalid_state'),
    issuer_mismatch: t('auth:sso.errors.issuer_mismatch'),
    jwks_failed: t('auth:sso.errors.jwks_failed'),
    jwks_missing: t('auth:sso.errors.jwks_missing'),
    missing_field: t('auth:sso.errors.missing_field'),
    missing_subject: t('auth:sso.errors.missing_subject'),
    no_admin: t('auth:sso.errors.no_admin'),
    no_email: t('auth:sso.errors.no_email'),
    saml_invalid: t('auth:sso.errors.saml_invalid'),
    signup_disabled: t('auth:sso.errors.signup_disabled'),
    sso_error: t('auth:sso.errors.sso_error'),
    token_exchange_failed: t('auth:sso.errors.token_exchange_failed'),
    user_missing: t('auth:sso.errors.user_missing'),
    userinfo_failed: t('auth:sso.errors.userinfo_failed'),
} satisfies Record<SsoErrorCode, string>)[code]

export const ssoPresetLabel = (t: TFunction, preset: SsoPresetKey): string => ({
    custom_oidc: t('forms:sso.presets.custom_oidc'),
    custom_saml: t('forms:sso.presets.custom_saml'),
} satisfies Record<SsoPresetKey, string>)[preset]

export const actionTypeLabel = (t: TFunction, type: ActionType): string => ({
    set_category: t('forms:automation.actionTypes.set_category'),
    add_tags: t('forms:automation.actionTypes.add_tags'),
    remove_tags: t('forms:automation.actionTypes.remove_tags'),
    set_description: t('forms:automation.actionTypes.set_description'),
    create_transfer: t('forms:automation.actionTypes.create_transfer'),
} satisfies Record<ActionType, string>)[type]

/** Action types in logs come from stored rules, so an unknown one stays as it is. */
export const actionTypeLabelLoose = (t: TFunction, type: string): string =>
    orRaw({
        set_category: t('forms:automation.actionTypes.set_category'),
        add_tags: t('forms:automation.actionTypes.add_tags'),
        remove_tags: t('forms:automation.actionTypes.remove_tags'),
        set_description: t('forms:automation.actionTypes.set_description'),
        create_transfer: t('forms:automation.actionTypes.create_transfer'),
    }, type)

export const conditionFieldLabel = (t: TFunction, field: ConditionField): string => ({
    type: t('forms:automation.fields.type'),
    amount: t('forms:automation.fields.amount'),
    description: t('forms:automation.fields.description'),
    account_id: t('forms:automation.fields.account_id'),
    category_id: t('forms:automation.fields.category_id'),
    tags: t('forms:automation.fields.tags'),
} satisfies Record<ConditionField, string>)[field]

export const operatorLabel = (t: TFunction, op: ConditionOperator): string => ({
    equals: t('forms:automation.operators.equals'),
    not_equals: t('forms:automation.operators.not_equals'),
    in: t('forms:automation.operators.in'),
    not_in: t('forms:automation.operators.not_in'),
    gt: t('forms:automation.operators.gt'),
    gte: t('forms:automation.operators.gte'),
    lt: t('forms:automation.operators.lt'),
    lte: t('forms:automation.operators.lte'),
    between: t('forms:automation.operators.between'),
    contains: t('forms:automation.operators.contains'),
    not_contains: t('forms:automation.operators.not_contains'),
    starts_with: t('forms:automation.operators.starts_with'),
    ends_with: t('forms:automation.operators.ends_with'),
    matches: t('forms:automation.operators.matches'),
    is_null: t('forms:automation.operators.is_null'),
    is_not_null: t('forms:automation.operators.is_not_null'),
    has_any: t('forms:automation.operators.has_any'),
    has_all: t('forms:automation.operators.has_all'),
    has_none: t('forms:automation.operators.has_none'),
} satisfies Record<ConditionOperator, string>)[op]

export const triggerLabel = (t: TFunction, trigger: TriggerType): string => ({
    on_transaction_create: t('forms:automation.triggers.on_transaction_create'),
    on_transaction_update: t('forms:automation.triggers.on_transaction_update'),
} satisfies Record<TriggerType, string>)[trigger]

export const triggerDescription = (t: TFunction, trigger: TriggerType): string => ({
    on_transaction_create: t('forms:automation.triggerDescriptions.on_transaction_create'),
    on_transaction_update: t('forms:automation.triggerDescriptions.on_transaction_update'),
} satisfies Record<TriggerType, string>)[trigger]

export const automationLogStatusLabel = (t: TFunction, status: AutomationLogStatus): string => ({
    success: t('pages:automation.logsStatus.success'),
    error: t('pages:automation.logsStatus.error'),
    skipped: t('pages:automation.logsStatus.skipped'),
} satisfies Record<AutomationLogStatus, string>)[status]

export const backupStatusLabel = (t: TFunction, status: BackupStatus): string => ({
    current: t('settings:backups.status.current'),
    unsigned: t('settings:backups.status.unsigned'),
    invalid: t('settings:backups.status.invalid'),
} satisfies Record<BackupStatus, string>)[status]

export const backupStatusHelp = (t: TFunction, status: BackupStatus): string => ({
    current: t('settings:backups.statusHelp.current'),
    unsigned: t('settings:backups.statusHelp.unsigned'),
    invalid: t('settings:backups.statusHelp.invalid'),
} satisfies Record<BackupStatus, string>)[status]

export const budgetPeriodLabel = (t: TFunction, period: BudgetPeriod): string => ({
    weekly: t('forms:budgets.periods.weekly'),
    monthly: t('forms:budgets.periods.monthly'),
    yearly: t('forms:budgets.periods.yearly'),
    one_time: t('forms:budgets.periods.one_time'),
} satisfies Record<BudgetPeriod, string>)[period]

/** Budget periods in lists come from the server, so an unknown one stays as it is. */
export const budgetPeriodLabelLoose = (t: TFunction, period: string): string =>
    orRaw({
        weekly: t('forms:budgets.periods.weekly'),
        monthly: t('forms:budgets.periods.monthly'),
        yearly: t('forms:budgets.periods.yearly'),
        one_time: t('forms:budgets.periods.one_time'),
    }, period)

export const categoryTypeLabel = (t: TFunction, type: CategoryType): string => ({
    income: t('pages:categories.types.income'),
    expense: t('pages:categories.types.expense'),
} satisfies Record<CategoryType, string>)[type]

export const debtTypeLabel = (t: TFunction, type: DebtType): string => ({
    i_owe: t('pages:debts.types.i_owe'),
    owed_to_me: t('pages:debts.types.owed_to_me'),
} satisfies Record<DebtType, string>)[type]

export const importStepLabel = (t: TFunction, step: ImportStep): string => ({
    upload: t('settings:import.steps.upload'),
    mapping: t('settings:import.steps.mapping'),
    preview: t('settings:import.steps.preview'),
    result: t('settings:import.steps.result'),
} satisfies Record<ImportStep, string>)[step]

export const frequencyLabel = (t: TFunction, frequency: RecurringFrequency): string => ({
    daily: t('forms:recurring.frequencies.daily'),
    weekly: t('forms:recurring.frequencies.weekly'),
    monthly: t('forms:recurring.frequencies.monthly'),
    yearly: t('forms:recurring.frequencies.yearly'),
} satisfies Record<RecurringFrequency, string>)[frequency]

/** Day of week, 0 = Sunday. */
export const weekdayLabel = (t: TFunction, day: number): string => [
    t('forms:recurring.weekdays.0'),
    t('forms:recurring.weekdays.1'),
    t('forms:recurring.weekdays.2'),
    t('forms:recurring.weekdays.3'),
    t('forms:recurring.weekdays.4'),
    t('forms:recurring.weekdays.5'),
    t('forms:recurring.weekdays.6'),
][day]

export const periodTypeLabel = (t: TFunction, type: PeriodType): string => ({
    last_30_days: t('pages:reports.filters.last_30_days'),
    month: t('pages:reports.filters.month'),
    quarter: t('pages:reports.filters.quarter'),
    year: t('pages:reports.filters.year'),
    ytd: t('pages:reports.filters.ytd'),
    custom: t('pages:reports.filters.custom'),
} satisfies Record<PeriodType, string>)[type]

export const reportTabLabel = (t: TFunction, tab: ReportTab): string => ({
    overview: t('pages:reports.tabs.overview'),
    cashflow: t('pages:reports.tabs.cashflow'),
    expenses: t('pages:reports.tabs.expenses'),
    income: t('pages:reports.tabs.income'),
    networth: t('pages:reports.tabs.networth'),
} satisfies Record<ReportTab, string>)[tab]

export const userRoleLabel = (t: TFunction, role: UserRole): string => ({
    admin: t('common:roles.admin'),
    user: t('common:roles.user'),
    guest: t('common:roles.guest'),
} satisfies Record<UserRole, string>)[role]

export const spaceRoleLabel = (t: TFunction, role: SpaceRole): string => ({
    admin: t('settings:spaces.roles.admin'),
    editor: t('settings:spaces.roles.editor'),
    viewer: t('settings:spaces.roles.viewer'),
} satisfies Record<SpaceRole, string>)[role]

export const transferReviewLabel = (t: TFunction, review: TransferReview): string => ({
    created_remote: t('settings:spaces.transfers.review.created_remote'),
    deleted_remote: t('settings:spaces.transfers.review.deleted_remote'),
    changed_remote: t('settings:spaces.transfers.review.changed_remote'),
    missing_remote: t('settings:spaces.transfers.review.missing_remote'),
} satisfies Record<TransferReview, string>)[review]

/** Audit actions are recorded by the server and may be newer than this client. */
export const auditActionLabel = (t: TFunction, action: string): string =>
    orRaw({
        assign_admin: t('settings:spaces.audit.actions.assign_admin'),
        set_quota: t('settings:spaces.audit.actions.set_quota'),
        delete_space: t('settings:spaces.audit.actions.delete_space'),
        restore_deleted_space: t('settings:spaces.audit.actions.restore_deleted_space'),
        promote_guest: t('settings:spaces.audit.actions.promote_guest'),
        trust_key: t('settings:spaces.audit.actions.trust_key'),
        untrust_key: t('settings:spaces.audit.actions.untrust_key'),
        rotate_key: t('settings:spaces.audit.actions.rotate_key'),
    }, action)

export const userTabLabel = (t: TFunction, tab: UserTab): string => ({
    preferences: t('settings:user.tabs.preferences'),
    security: t('settings:user.tabs.security'),
    api: t('settings:user.tabs.api'),
} satisfies Record<UserTab, string>)[tab]

export const sortOptionLabel = (t: TFunction, option: SortOption): string => ({
    dateNewest: t('pages:transactions.sort.dateNewest'),
    dateOldest: t('pages:transactions.sort.dateOldest'),
    amountHigh: t('pages:transactions.sort.amountHigh'),
    amountLow: t('pages:transactions.sort.amountLow'),
} satisfies Record<SortOption, string>)[option]

export const transactionTypeLabel = (t: TFunction, type: TransactionType): string => ({
    income: t('pages:transactions.types.income'),
    expense: t('pages:transactions.types.expense'),
    transfer: t('pages:transactions.types.transfer'),
    debt_payment: t('pages:transactions.types.debt_payment'),
    debt_collection: t('pages:transactions.types.debt_collection'),
    debt_lend: t('pages:transactions.types.debt_lend'),
    debt_borrow: t('pages:transactions.types.debt_borrow'),
    transfer_out: t('pages:transactions.types.transfer_out'),
    transfer_in: t('pages:transactions.types.transfer_in'),
} satisfies Record<TransactionType, string>)[type]

export type NavKey =
    | 'dashboard' | 'transactions' | 'recurring' | 'budgets' | 'debts' | 'reports'
    | 'accounts' | 'categories' | 'tags' | 'currencies' | 'automation' | 'space' | 'user' | 'import'
    | 'backups' | 'system' | 'monitoring' | 'sso' | 'security' | 'systemBackups' | 'users' | 'spaces'

export const navLabel = (t: TFunction, key: NavKey): string => ({
    dashboard: t('nav:dashboard'),
    transactions: t('nav:transactions'),
    recurring: t('nav:recurring'),
    budgets: t('nav:budgets'),
    debts: t('nav:debts'),
    reports: t('nav:reports'),
    accounts: t('nav:accounts'),
    categories: t('nav:categories'),
    tags: t('nav:tags'),
    currencies: t('nav:currencies'),
    automation: t('nav:automation'),
    space: t('nav:space'),
    user: t('nav:user'),
    import: t('nav:import'),
    backups: t('nav:backups'),
    system: t('nav:system'),
    monitoring: t('nav:monitoring'),
    sso: t('nav:sso'),
    security: t('nav:security'),
    systemBackups: t('nav:systemBackups'),
    users: t('nav:users'),
    spaces: t('nav:spaces'),
} satisfies Record<NavKey, string>)[key]

/** Entries of the "new transaction" menu; `transfer_out` is a transfer to another space. */
export const createMenuLabel = (t: TFunction, type: 'income' | 'expense' | 'transfer' | 'transfer_out'): string => ({
    income: t('nav:income'),
    expense: t('nav:expense'),
    transfer: t('nav:transfer'),
    transfer_out: t('nav:transfer_out'),
})[type]

/** Transaction types from imported or server data, which may be ones this client does not know. */
export const transactionTypeLabelLoose = (t: TFunction, type: string): string =>
    orRaw({
        income: t('pages:transactions.types.income'),
        expense: t('pages:transactions.types.expense'),
        transfer: t('pages:transactions.types.transfer'),
        debt_payment: t('pages:transactions.types.debt_payment'),
        debt_collection: t('pages:transactions.types.debt_collection'),
        debt_lend: t('pages:transactions.types.debt_lend'),
        debt_borrow: t('pages:transactions.types.debt_borrow'),
        transfer_out: t('pages:transactions.types.transfer_out'),
        transfer_in: t('pages:transactions.types.transfer_in'),
    }, type)
