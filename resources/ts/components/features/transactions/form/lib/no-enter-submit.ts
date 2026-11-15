import type { KeyboardEvent } from 'react'

/**
 * For the `onKeyDown` of the transaction form: Enter in a text field does not submit it. The form is too big to be
 * sent by a key pressed in passing; it is saved with its button. Enter still does its own work elsewhere: it opens a
 * select, presses a button, adds a tag, and works out a formula in an amount (that field handles it first).
 */
export function noEnterSubmit(event: KeyboardEvent<HTMLFormElement>): void {
    if (event.key === 'Enter' && event.target instanceof HTMLInputElement) {
        event.preventDefault()
    }
}
