import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import { schemaResolver } from '@/lib/form-resolver'
import { Form } from '@/components/ui/form'
import { getDebtSchema, DebtFormData } from '@/schemas'
import { useCurrencies, useAccounts, useFormValuesChange } from '@/hooks'
import { Banknote, HandCoins } from 'lucide-react'
import {
    FormInputField,
    FormSubmitButton,
    FormWrapper,
    SegmentedChoice,
    useNegativeBalanceConfirm,
} from '@/components/shared'
import { warningsForAccountOutflow } from '@/lib/negative-balance'
import { cn, formatDateLocal } from '@/lib/utils'
import { debtTypeLabel } from '@/lib/labels'
import { DebtAmountFields } from './form/DebtAmountFields'
import { DebtOriginField } from './form/DebtOriginField'

interface DebtFormProps {
    defaultValues?: Partial<DebtFormData>
    onSubmit: (data: DebtFormData) => void
    onValuesChange?: (data: DebtFormData) => void
    isSubmitting?: boolean
    submitLabel?: string
    formId?: string
    hideSubmit?: boolean
    mode?: 'create' | 'edit'
}

const DEBT_TYPES = [
    {
        value: 'i_owe',
        icon: Banknote,
        color: 'text-destructive',
    },
    {
        value: 'owed_to_me',
        icon: HandCoins,
        color: 'text-success',
    },
] as const

function today(): string {
    return formatDateLocal()
}

export function DebtForm({
    defaultValues,
    onSubmit,
    onValuesChange,
    isSubmitting,
    submitLabel,
    formId,
    hideSubmit,
    mode = 'create',
}: DebtFormProps) {
    const { t } = useTranslation(['common', 'forms', 'pages'])
    const { data: currencies } = useCurrencies()
    const { data: accounts } = useAccounts({ active: true, exclude_debts: true })
    const { confirmIfNeeded, dialog: negativeBalanceDialog } = useNegativeBalanceConfirm<DebtFormData>()

    const form = useForm<DebtFormData>({
        resolver: schemaResolver<DebtFormData>(getDebtSchema(mode)),
        defaultValues: {
            origin: 'new',
            name: '',
            debt_type: 'i_owe',
            currency_id: 0,
            account_id: 0,
            date: today(),
            amount: 0,
            due_date: '',
            counterparty: '',
            description: '',
            ...defaultValues,
        },
    })

    const origin = form.watch('origin') ?? 'new'
    const debtType = form.watch('debt_type')
    const accountId = form.watch('account_id')
    const currencyId = form.watch('currency_id')
    const amount = Number(form.watch('amount')) || 0
    const selectedAccount = accounts?.find((account) => account.id === Number(accountId))
    const selectedCurrency = currencies?.find((currency) => currency.id === Number(currencyId))
    const insufficientFunds =
        mode === 'create' &&
        origin === 'new' &&
        debtType === 'owed_to_me' &&
        !!selectedAccount &&
        amount > selectedAccount.currentBalance

    useFormValuesChange(form, onValuesChange)

    const showAccount = mode === 'create' && origin === 'new'
    const showCurrency = mode === 'edit' || origin === 'existing'
    const amountCurrencySymbol = showAccount ? selectedAccount?.currency?.symbol : selectedCurrency?.symbol

    return (
        <FormWrapper>
            <Form {...form}>
                <form
                    id={formId}
                    onSubmit={form.handleSubmit((data) => {
                        const warnings =
                            showAccount && debtType === 'owed_to_me'
                                ? warningsForAccountOutflow(selectedAccount, amount)
                                : []
                        confirmIfNeeded(data, warnings, onSubmit)
                    })}
                    className="space-y-4"
                >
                    <SegmentedChoice
                        value={debtType}
                        onChange={(value) => form.setValue('debt_type', value)}
                        options={DEBT_TYPES.map((option) => ({
                            ...option,
                            label: debtTypeLabel(t, option.value),
                        }))}
                    />

                    <FormInputField
                        control={form.control}
                        name="name"
                        label={t('fields.name')}
                        inputProps={{ placeholder: t('forms:debts.namePlaceholder') }}
                    />

                    {mode === 'create' && <DebtOriginField control={form.control} />}

                    <DebtAmountFields
                        control={form.control}
                        debtType={debtType}
                        showAccount={showAccount}
                        showCurrency={showCurrency}
                        currencySymbol={amountCurrencySymbol}
                        insufficientFundsAccount={insufficientFunds ? selectedAccount : undefined}
                    />

                    <FormInputField
                        control={form.control}
                        name="counterparty"
                        label={t('forms:debts.counterparty')}
                        description={t('forms:debts.counterpartyHelp')}
                        inputProps={{ placeholder: t('forms:debts.counterpartyPlaceholder') }}
                    />

                    <div className={cn('grid gap-4', showAccount && 'sm:grid-cols-2')}>
                        {showAccount && (
                            <FormInputField
                                control={form.control}
                                name="date"
                                label={t('forms:debts.issuedDate')}
                                inputProps={{ type: 'date' }}
                            />
                        )}
                        <FormInputField
                            control={form.control}
                            name="due_date"
                            label={t('forms:debts.dueDate')}
                            inputProps={{ type: 'date' }}
                        />
                    </div>

                    <FormInputField
                        control={form.control}
                        name="description"
                        label={t('fields.description')}
                        multiline
                        textareaProps={{ placeholder: t('forms:debts.notesPlaceholder') }}
                    />

                    <FormSubmitButton hidden={hideSubmit} isSubmitting={isSubmitting} label={submitLabel} />
                </form>
            </Form>
            {negativeBalanceDialog}
        </FormWrapper>
    )
}
