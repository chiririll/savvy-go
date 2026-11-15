// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { afterEach, expect, it, vi } from 'vitest'
import { currencies } from '@/test/fixtures'
import { AmountInput } from './AmountInput'
import { noEnterSubmit } from '../lib/no-enter-submit'

afterEach(cleanup)

function Form({ onSubmit }: { onSubmit: () => void }) {
    const [value, setValue] = useState('')
    return (
        <form onKeyDown={noEnterSubmit} onSubmit={(e) => { e.preventDefault(); onSubmit() }}>
            <AmountInput aria-label="amount" currency={currencies[0]} value={value} onChange={setValue} />
            <button type="submit">save</button>
        </form>
    )
}

it('Enter works a formula out and submits nothing', async () => {
    const onSubmit = vi.fn()
    render(<Form onSubmit={onSubmit} />)
    const user = userEvent.setup()
    await user.type(screen.getByLabelText('amount'), '2+3{Enter}')
    expect((screen.getByLabelText('amount') as HTMLInputElement).value).toBe('5.00')
    expect(onSubmit).not.toHaveBeenCalled()
    await user.keyboard('{Enter}')
    expect(onSubmit).not.toHaveBeenCalled()
})
