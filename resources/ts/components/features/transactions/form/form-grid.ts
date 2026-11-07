/**
 * A row of the income and expense form: two halves of the same width. The amount and the date are the first, the
 * account and the category the second; the button of a row (remove, clear, add a line) sits inside its half and
 * takes room from the field, so the halves never change.
 */
export const FORM_ROW = 'flex min-w-0 items-start gap-x-2 sm:gap-x-4 [&>*]:min-w-0 [&>*]:flex-1 [&>*]:basis-0'
