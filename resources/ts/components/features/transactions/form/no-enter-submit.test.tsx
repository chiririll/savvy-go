// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'
import { noEnterSubmit } from './no-enter-submit'

afterEach(cleanup)

function Form({ onSubmit }: { onSubmit: () => void }) {
    return (
        <form onKeyDown={noEnterSubmit} onSubmit={(e) => { e.preventDefault(); onSubmit() }}>
            <input aria-label="text" />
            <textarea aria-label="note" />
            <button type="submit">save</button>
        </form>
    )
}

it('does not submit on Enter in a text field', async () => {
    const onSubmit = vi.fn()
    render(<Form onSubmit={onSubmit} />)
    await userEvent.setup().type(screen.getByLabelText('text'), '12{Enter}')
    expect(onSubmit).not.toHaveBeenCalled()
})

it('keeps a new line in a textarea and the button working', async () => {
    const onSubmit = vi.fn()
    render(<Form onSubmit={onSubmit} />)
    const user = userEvent.setup()
    await user.type(screen.getByLabelText('note'), 'a{Enter}b')
    expect((screen.getByLabelText('note') as HTMLTextAreaElement).value).toBe('a\nb')
    await user.click(screen.getByText('save'))
    expect(onSubmit).toHaveBeenCalledOnce()
    onSubmit.mockClear()
    screen.getByText('save').focus()
    await user.keyboard('{Enter}')
    expect(onSubmit).toHaveBeenCalledOnce()
})
