// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FormProvider, useFieldArray, useForm } from 'react-hook-form'
import { afterEach, expect, it, vi } from 'vitest'
import type { TransactionFormValues } from '@/schemas/transactions'
import { currencies } from '@/test/fixtures'
import { noEnterSubmit } from '../lib/no-enter-submit'
import { TransactionLines } from './TransactionLines'

vi.mock('@/components/shared', () => ({ CategorySelect: () => <span /> }))

afterEach(cleanup)

function Harness({ onSubmit }: { onSubmit: () => void }) {
    const form = useForm<TransactionFormValues>({
        defaultValues: { lines: [{ category_id: 1, currency_id: null, amount: '10' as unknown as number, note: '' }] } as TransactionFormValues,
    })
    const lines = useFieldArray({ control: form.control, name: 'lines' })
    return (
        <FormProvider {...form}>
            <form onKeyDown={noEnterSubmit} onSubmit={(e) => { e.preventDefault(); onSubmit() }}>
                <TransactionLines
                    control={form.control}
                    fields={lines.fields}
                    remove={lines.remove}
                    type="expense"
                    currencies={[currencies[0]]}
                    coverage={{ currency: currencies[0], total: 10, remainder: 0, excess: 0 }}
                />
                <button type="submit">save</button>
            </form>
        </FormProvider>
    )
}

it('Enter in the amount of the only line evaluates the formula and does not submit', async () => {
    const onSubmit = vi.fn()
    render(<Harness onSubmit={onSubmit} />)
    const user = userEvent.setup()
    const box = screen.getByLabelText('Amount') as HTMLInputElement
    await user.clear(box)
    await user.type(box, '2+3{Enter}')
    expect(box.value).toBe('5.00')
    expect(onSubmit).not.toHaveBeenCalled()
})
