import type { RecurringFormValues } from '@/schemas/recurring'
import type {
    Account,
    Category,
    Currency,
    LineInput,
    PaymentInput,
    RecurringInput,
    RecurringSchedule,
    RecurringTemplate,
    RecurringTransaction,
    Transaction,
    TransactionLine,
} from '@/types'
import { canEditInForm, toFormValues } from './transaction-form-values'
import {
    buildTransactionInput,
    roundToCurrency,
    toBase,
    type PayloadError,
} from './transaction-payload'
import { buildTransferInput, type TransferError } from './transfer-payload'

/**
 * A rule keeps the transaction it makes as the JSON it was saved with. Another client may have saved an amount as
 * a string, so every amount is read as a number; no amount (`null`) is a payment the occurrence works out.
 */
const amountOf = (value: unknown): number | null => (value === null || value === undefined || value === '' ? null : Number(value))

const accountOf = (accounts: Account[], id: unknown): Account | undefined => accounts.find((a) => a.id === Number(id))

/** Whether the amount is only a guess: marked as an estimate, or left for the rates of each day to decide. */
export function isApproximateRule(template: RecurringTemplate): boolean {
    return template.payments.some((payment) => payment.estimated || amountOf(payment.amount) === null)
}

/**
 * Whether an occurrence made by a rule shows a guess: it is still pending, and its rule's amount is an estimate or
 * left for the rates of the day. A confirmed occurrence has its real amount. `approximateRules` holds the ids of
 * the rules whose template is approximate.
 */
export function isApproximateOccurrence(
    transaction: Pick<Transaction, 'status' | 'recurringTransactionId'>,
    approximateRules: ReadonlySet<number>,
): boolean {
    return (
        transaction.status === 'pending' &&
        transaction.recurringTransactionId !== null &&
        approximateRules.has(transaction.recurringTransactionId)
    )
}

/**
 * What the payment without an amount takes at today's rates: whatever makes the payments move what the lines
 * account for, as the API works it out for each occurrence. Null when nothing is open.
 */
function balancingPayment(template: RecurringTemplate, accounts: Account[], linesInBase: number): { index: number; amount: number } | null {
    const index = template.payments.findIndex((payment) => amountOf(payment.amount) === null)
    if (index < 0) {
        return null
    }
    const open = accountOf(accounts, template.payments[index].account_id)?.currency
    if (!open || !(toBase(open) > 0)) {
        return null
    }
    const others = template.payments.reduce((sum, payment, i) => {
        const currency = accountOf(accounts, payment.account_id)?.currency
        const amount = amountOf(payment.amount)
        return i === index || amount === null || !currency ? sum : sum + amount * toBase(currency)
    }, 0)
    const target = template.type === 'income' ? linesInBase : -linesInBase
    return { index, amount: roundToCurrency((target - others) / toBase(open), open) }
}

/** The currency of a line: the one it names, else the one all the payment accounts share. */
function lineCurrencyOf(line: LineInput, payments: PaymentInput[], accounts: Account[]): Currency | null {
    const currencies = payments.flatMap((p) => accountOf(accounts, p.account_id)?.currency ?? [])
    const named = line.currency_id ? currencies.find((c) => c.id === Number(line.currency_id)) : undefined
    if (named) {
        return named
    }
    const shared = currencies.filter((c) => c.id === currencies[0]?.id)
    return !line.currency_id && currencies.length > 0 && shared.length === currencies.length ? currencies[0] : null
}

/**
 * A rule's template as the transaction it makes, so the rows and the form read it the way they read any other:
 * accounts and categories filled in, an amount left open worked out from today's rates. Null when an account
 * of the template is gone, or a line has no currency to be read in.
 */
export function templateAsTransaction(
    rule: Pick<RecurringTransaction, 'id' | 'template'>,
    accounts: Account[],
    categories: Category[] = [],
): Transaction | null {
    const { template } = rule
    if (template.payments.some((p) => !accountOf(accounts, p.account_id)?.currency)) {
        return null
    }

    const lines: TransactionLine[] = []
    let linesInBase = 0
    for (const [seq, line] of (template.lines ?? []).entries()) {
        const currency = lineCurrencyOf(line, template.payments, accounts)
        if (!currency) {
            return null
        }
        const amount = Number(line.amount)
        linesInBase += amount * toBase(currency)
        lines.push({
            seq,
            categoryId: line.category_id ?? null,
            category: categories.find((c) => c.id === Number(line.category_id)) ?? null,
            currencyId: currency.id,
            amount,
            note: line.note ?? null,
        })
    }

    const open = balancingPayment(template, accounts, linesInBase)
    const payments = template.payments.map((p, index) => ({
        accountId: Number(p.account_id),
        account: accountOf(accounts, p.account_id) ?? null,
        // A template that cannot work an open amount out shows it as nothing, which the form refuses to open.
        amount: open?.index === index ? open.amount : (amountOf(p.amount) ?? 0),
    }))

    return {
        id: rule.id,
        type: template.type,
        description: template.description ?? null,
        date: null,
        status: 'pending',
        recurringTransactionId: rule.id,
        amount: null,
        rowAmount: null,
        receivedAmount: null,
        difference: null,
        actions: { edit: false, editDetails: false, duplicate: false, delete: false, confirm: false, skip: false },
        restrictions: [],
        payments,
        lines: lines ?? [],
        tags: [],
    }
}

/**
 * Whether the form can edit a rule without losing anything: what it makes could be edited as a transaction, and
 * the only amount it leaves open is a transfer's amount received (the form asks for every other amount).
 */
export function canEditRecurringInForm(rule: RecurringTransaction, accounts: Account[]): boolean {
    const { template } = rule
    const open = template.payments.filter((p) => amountOf(p.amount) === null)
    if (open.length > 1 || (open.length === 1 && template.type !== 'transfer')) {
        return false
    }
    const tx = templateAsTransaction(rule, accounts)
    // The open side of a transfer is the one that receives.
    if (open.length === 1 && tx && !(tx.payments[template.payments.indexOf(open[0])].amount > 0)) {
        return false
    }
    return tx !== null && canEditInForm(tx)
}

const parts = (date: string) => {
    const [year, month, day] = date.split('-').map(Number)
    return { weekday: new Date(year, month - 1, day).getDay(), day }
}

/** The schedule as the form holds it: a weekly rule without a day takes the weekday of its start, a monthly one its day. */
function scheduleValues(schedule: RecurringSchedule) {
    const start = parts(schedule.start)
    return {
        frequency: schedule.frequency,
        interval: schedule.interval,
        day_of_week: schedule.day_of_week ?? start.weekday,
        day_of_month: schedule.day_of_month ?? start.day,
        start_date: schedule.start,
        end_date: schedule.end ?? null,
    }
}

/** The form values for a rule `canEditRecurringInForm` accepts; null for one it does not. */
export function recurringToFormValues(rule: RecurringTransaction, accounts: Account[]): RecurringFormValues | null {
    if (!canEditRecurringInForm(rule, accounts)) {
        return null
    }
    const tx = templateAsTransaction(rule, accounts)!
    const base = toFormValues(tx)
    const open = rule.template.payments.some((p) => amountOf(p.amount) === null)

    return {
        ...base,
        type: base.type as RecurringFormValues['type'],
        date: null,
        tag_ids: rule.template.tag_ids ?? [],
        // The open amount was worked out to read the shape; the form keeps it open.
        ...(open ? { to_amount: null, exchange_rate: null } : {}),
        estimated: rule.template.payments.some((p) => p.estimated),
        is_active: rule.isActive,
        ...scheduleValues(rule.schedule),
    }
}

/** The schedule the API takes: it refuses a day that does not fit the frequency, so only the one in use is sent. */
function scheduleOf(values: RecurringFormValues): RecurringSchedule {
    return {
        frequency: values.frequency,
        interval: Number(values.interval),
        ...(values.frequency === 'weekly' && values.day_of_week != null ? { day_of_week: Number(values.day_of_week) } : {}),
        ...(values.frequency === 'monthly' && values.day_of_month != null ? { day_of_month: Number(values.day_of_month) } : {}),
        start: values.start_date,
        end: values.end_date ? values.end_date : null,
    }
}

export type RecurringBuildResult =
    | { ok: true; input: RecurringInput }
    | { ok: false; source: 'payload'; error: PayloadError }
    | { ok: false; source: 'transfer'; error: TransferError }

/** The rule the API takes for what the form holds: the template has no date, as an occurrence is dated by its day. */
export function buildRecurringInput(values: RecurringFormValues, accounts: Account[]): RecurringBuildResult {
    const own = { ...values, date: null }
    const built =
        values.type === 'transfer'
            ? buildTransferInput({ ...own, type: 'transfer' }, accounts, { template: true })
            : buildTransactionInput({ ...own, type: values.type }, accounts)
    if (!built.ok) {
        return values.type === 'transfer'
            ? { ok: false, source: 'transfer', error: built.error as TransferError }
            : { ok: false, source: 'payload', error: built.error as PayloadError }
    }

    const { description, payments, lines, tag_ids } = built.input
    const template: RecurringTemplate = {
        type: values.type,
        description,
        // An amount left open is a guess by nature: the estimate is said so, as the API's own example does.
        payments: (payments ?? []).map((p) => (values.estimated || p.amount === null ? { ...p, estimated: true } : p)),
        lines: lines ?? [],
        tag_ids,
    }

    return { ok: true, input: { template, schedule: scheduleOf(values), is_active: values.is_active } }
}
