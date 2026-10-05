import { useForm, useWatch } from 'react-hook-form'
import { schemaResolver } from '@/lib/form-resolver'
import { useEffect, useLayoutEffect, useMemo, useRef, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import {
    Form,
    FormField,
    FormItem,
    FormLabel,
    FormControl,
    FormMessage,
} from '@/components/ui/form'
import { getTransactionSchema, isTransferType, TransactionFormValues } from '@/schemas/transactions'
import {
    useAccounts,
    useCategories,
    useCurrencies,
    useFormValuesChange,
    useTransferTargets,
    useTransactionPartyDefaults,
} from '@/hooks'
import { formatDateLocal, isDateInFuture } from '@/lib/utils'
import { currencyDecimals, sumTransactionItems } from '@/lib/transaction-items'
import {
    CategorySelect,
    FormEstimatedField,
    FormWrapper,
    MoneyAccountFields,
    TagSelect,
    TransactionTypeTabs,
    useNegativeBalanceConfirm,
} from '@/components/shared'
import { TransactionFormItems } from './TransactionFormItems'
import { toBalanceImpact, useBalancePreview } from './form/balance-preview'
import { resolveOpeningTransferRate, useTransferRate } from './form/transfer-rate'
import { SpaceTransferTargetField } from './form/SpaceTransferTargetField'
import { TransactionBalanceHint } from './form/TransactionBalanceHint'
import { TransactionDateField } from './form/TransactionDateField'
import { TransferRateField } from './form/TransferRateField'

const ACCOUNT_FILTER = { active: true, exclude_debts: true }

interface TransactionFormProps {
    defaultValues?: Partial<TransactionFormValues>
    onSubmit: (data: TransactionFormValues) => void
    onTypeChange?: (type: TransactionFormValues['type']) => void
    onValuesChange?: (data: TransactionFormValues) => void
    isSubmitting?: boolean
    submitLabel?: string
    formId?: string
    hideSubmit?: boolean
    isEdit?: boolean
    originalAffectsBalance?: boolean
    onPreviewChange?: (preview: ReactNode) => void
    open?: boolean
}

export function TransactionForm({
    defaultValues,
    onSubmit,
    onTypeChange,
    onValuesChange,
    isSubmitting,
    submitLabel,
    formId,
    hideSubmit,
    isEdit,
    originalAffectsBalance,
    onPreviewChange,
    open = true,
}: TransactionFormProps) {
    const { t } = useTranslation(['common', 'forms'])
    const { data: accounts } = useAccounts(ACCOUNT_FILTER)
    const { data: categories } = useCategories()
    const { data: currencies } = useCurrencies()
    const transferTargets = useTransferTargets()
    const { confirmIfNeeded, dialog: negativeBalanceDialog } = useNegativeBalanceConfirm<TransactionFormValues>()

    const formDefaults = useMemo(() => ({
        type: defaultValues?.type ?? 'expense' as const,
        account_id: defaultValues?.account_id ?? 0,
        to_account_id: defaultValues?.to_account_id ?? null,
        to_space_id: defaultValues?.to_space_id ?? null,
        category_id: defaultValues?.category_id ?? null,
        amount: defaultValues?.amount ?? 0,
        to_amount: defaultValues?.to_amount ?? null,
        is_estimated: defaultValues?.is_estimated ?? false,
        exchange_rate: resolveOpeningTransferRate(defaultValues),
        description: defaultValues?.description ?? '',
        date: defaultValues?.date !== undefined ? (defaultValues.date ?? '') : formatDateLocal(),
        items: defaultValues?.items ?? [],
        tag_ids: defaultValues?.tag_ids ?? [],
    }), [defaultValues])

    const schemaOptionsRef = useRef({
        rejectFutureDate: Boolean(originalAffectsBalance),
        currencyDecimals: 2,
    })

    const form = useForm<TransactionFormValues>({
        resolver: (values, context, options) => schemaResolver<TransactionFormValues>(
            getTransactionSchema(schemaOptionsRef.current),
        )(values, context, options),
        defaultValues: formDefaults,
    })

    // Runs before the effects below, so the defaults they fill in survive it.
    const formDefaultsRef = useRef(formDefaults)
    formDefaultsRef.current = formDefaults
    useEffect(() => {
        if (open) {
            form.reset(formDefaultsRef.current)
        }
    }, [open, form])

    useFormValuesChange(
        form,
        onValuesChange
            ? (data) => onValuesChange({ ...data, exchange_rate: null })
            : undefined,
    )

    const [transactionType, accountId, toAccountId, toSpaceId, items, date] = useWatch({
        control: form.control,
        name: ['type', 'account_id', 'to_account_id', 'to_space_id', 'items', 'date'],
    })
    const isTransfer = isTransferType(transactionType)
    const isSpaceTransfer = transactionType === 'transfer_out'
    const hasItems = Boolean(items?.length)
    // The transfers API that stores a transfer to another space needs a date.
    const dateRequired = Boolean(originalAffectsBalance) || isSpaceTransfer
    const isPendingDate = !date || isDateInFuture(date)

    // Where the destination account lives: undefined is this space.
    const destinationSpaceId = isSpaceTransfer ? Number(toSpaceId) || null : undefined
    const { data: destinationSpaceAccounts } = useAccounts(ACCOUNT_FILTER, destinationSpaceId ?? null)

    useTransactionPartyDefaults(form, accounts, categories)

    useEffect(() => {
        if (isTransfer) {
            form.setValue('category_id', null)
        }
    }, [isTransfer, form])

    const selectedAccount = accounts?.find(a => a.id === Number(accountId))
    const selectedToAccount = (isSpaceTransfer ? destinationSpaceAccounts : accounts)
        ?.find(a => a.id === Number(toAccountId))
    const itemDecimals = currencyDecimals(selectedAccount?.currency)
    schemaOptionsRef.current = {
        rejectFutureDate: Boolean(originalAffectsBalance),
        currencyDecimals: itemDecimals,
    }

    const itemsTotal = sumTransactionItems(items, itemDecimals)
    useEffect(() => {
        if (hasItems && itemsTotal > 0) {
            form.setValue('amount', itemsTotal, { shouldValidate: false })
        }
    }, [itemsTotal, hasItems, form])

    const rate = useTransferRate({
        form,
        from: selectedAccount,
        to: selectedToAccount,
        // Rates are relative to this space's base currency, so look the destination up by code.
        toRate: currencies?.find(c => c.code === selectedToAccount?.currency?.code)?.rate,
        open,
    })

    const posted = useMemo(
        () => (isEdit && originalAffectsBalance && defaultValues ? toBalanceImpact(defaultValues) : null),
        [isEdit, originalAffectsBalance, defaultValues],
    )
    const balances = useBalancePreview({
        control: form.control,
        from: selectedAccount,
        to: selectedToAccount,
        posted,
        willPost: Boolean(originalAffectsBalance) || (!isEdit && !isPendingDate),
    })

    const balanceHint = (
        <TransactionBalanceHint
            from={balances.from}
            to={balances.to}
            isPending={isPendingDate}
            isUndated={!date}
        />
    )

    useLayoutEffect(() => {
        onPreviewChange?.(balanceHint)
    }, [onPreviewChange, balances, isPendingDate, date])

    return (
        <FormWrapper>
        <Form {...form}>
            <form
                id={formId}
                noValidate
                onSubmit={form.handleSubmit((data) => {
                    if (dateRequired && !data.date) {
                        form.setError('date', { message: t('validation.dateRequired') })
                        return
                    }
                    rate.commit()
                    confirmIfNeeded(
                        {
                            ...data,
                            to_amount: form.getValues('to_amount'),
                            // UI-only helper; the server derives it from the amounts.
                            exchange_rate: undefined,
                        },
                        balances.warnings,
                        onSubmit,
                    )
                })}
                className="space-y-6"
            >
                <TransactionTypeTabs
                    value={transactionType}
                    withSpaceTransfer={!isEdit && transferTargets.length > 0}
                    onChange={(value) => {
                        if ((value === 'transfer_out') !== isSpaceTransfer) {
                            // The destination account belongs to another space now.
                            form.setValue('to_account_id', null)
                        }
                        form.setValue('type', value)
                        onTypeChange?.(value)
                    }}
                />

                {!onPreviewChange && balanceHint}

                {isSpaceTransfer && (
                    <SpaceTransferTargetField
                        links={transferTargets}
                        noAccounts={destinationSpaceAccounts?.length === 0}
                    />
                )}

                <MoneyAccountFields
                    control={form.control}
                    isTransfer={isTransfer}
                    toSpaceId={destinationSpaceId}
                    accountId={accountId}
                    fromCurrencySymbol={selectedAccount?.currency?.symbol}
                    toCurrencySymbol={selectedToAccount?.currency?.symbol}
                    amountDisabled={hasItems}
                    toAmountReadOnly={rate.sameCurrency}
                    onSwapAccounts={rate.keepNextPair}
                />

                {(isPendingDate || (isEdit && !originalAffectsBalance)) && !hasItems && !isSpaceTransfer && (
                    <FormEstimatedField
                        control={form.control}
                        help={t('forms:transactions.estimatedHelp')}
                    />
                )}

                <div className="grid min-w-0 grid-cols-1 gap-4 sm:grid-cols-2">
                    {isTransfer ? (
                        <TransferRateField
                            rate={rate}
                            fromCode={selectedAccount?.currency?.code}
                            toCode={selectedToAccount?.currency?.code}
                        />
                    ) : (
                        <FormField
                            control={form.control}
                            name="category_id"
                            render={({ field }) => (
                                <FormItem className="min-w-0">
                                    <FormLabel>{t('fields.category')}</FormLabel>
                                    <CategorySelect
                                        value={field.value}
                                        onChange={field.onChange}
                                        type={transactionType as 'income' | 'expense'}
                                    />
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    )}

                    <TransactionDateField
                        required={dateRequired}
                        max={originalAffectsBalance ? formatDateLocal() : undefined}
                    />
                </div>

                <FormField
                    control={form.control}
                    name="description"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{t('fields.description')}</FormLabel>
                            <FormControl>
                                <Textarea
                                    placeholder={t('forms:transactions.notesPlaceholder')}
                                    className="resize-none h-20"
                                    {...field}
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                {/* The transfers API keeps no tags. */}
                {!isSpaceTransfer && (
                    <FormField
                        control={form.control}
                        name="tag_ids"
                        render={({ field }) => (
                            <TagSelect
                                value={field.value ?? []}
                                onChange={field.onChange}
                                asFormItem
                            />
                        )}
                    />
                )}

                {!isTransfer && (
                    <TransactionFormItems form={form} currency={selectedAccount?.currency} />
                )}

                {!hideSubmit && (
                    <Button type="submit" disabled={isSubmitting} className="w-full">
                        {isSubmitting ? t('actions.saving') : (submitLabel ?? t('actions.save'))}
                    </Button>
                )}
            </form>
        </Form>
        {negativeBalanceDialog}
        </FormWrapper>
    )
}
