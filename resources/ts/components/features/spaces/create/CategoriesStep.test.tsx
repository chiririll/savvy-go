// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { CategoriesStep } from './CategoriesStep'
import type { DraftContent } from './draft'

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key, i18n: { language: 'en' } }),
}))

const content: DraftContent = {
    baseCurrency: 'USD',
    currencies: [],
    accounts: [],
    categories: [
        { name: 'Groceries', type: 'expense', icon: '🛒', color: '#16a34a', isDefault: false },
        { name: 'Salary', type: 'income', icon: '💰', color: '#2563eb', isDefault: false },
        { name: 'Rent', type: 'expense', icon: '🏠', color: '#16a34a', isDefault: false },
    ],
    tags: [],
}

function Harness({ initial = content }: { initial?: DraftContent }) {
    const [state, setState] = useState(initial)
    return <CategoriesStep content={state} onChange={(patch) => setState((c) => ({ ...c, ...patch }))} />
}

afterEach(() => cleanup())

describe('CategoriesStep groups', () => {
    it('lists the categories under a heading per type, in the order of the draft', () => {
        render(<Harness />)
        const expenses = screen.getByTestId('create-space-categories-expense')
        const incomes = screen.getByTestId('create-space-categories-income')
        expect(within(expenses).getByRole('heading').textContent).toBe('spaces.create.categories.groupExpense')
        expect(within(incomes).getByRole('heading').textContent).toBe('spaces.create.categories.groupIncome')
        expect(within(expenses).getAllByRole('textbox').map((i) => (i as HTMLInputElement).value)).toEqual([
            'Groceries',
            'Rent',
        ])
        expect(within(incomes).getAllByRole('textbox').map((i) => (i as HTMLInputElement).value)).toEqual(['Salary'])
    })

    it('leaves out the heading of a type with no categories', () => {
        render(<Harness initial={{ ...content, categories: [content.categories[0]] }} />)
        expect(screen.queryByTestId('create-space-categories-income')).toBeNull()
    })

    it('does not repeat the type on each row', () => {
        render(<Harness />)
        for (const row of screen.getAllByRole('listitem')) {
            expect(within(row).queryByText(/^pages:categories\.types\./)).toBeNull()
        }
    })

    it('renames and removes the right category inside a group', () => {
        render(<Harness />)
        fireEvent.change(screen.getAllByRole('textbox', { name: 'spaces.create.categories.name' })[2], {
            target: { value: 'Pay' },
        })
        expect(screen.getByDisplayValue('Pay')).toBeTruthy()
        expect(screen.queryByDisplayValue('Salary')).toBeNull()
        const incomes = screen.getByTestId('create-space-categories-income')
        fireEvent.click(within(incomes).getByRole('button', { name: 'spaces.create.categories.remove' }))
        expect(screen.queryByTestId('create-space-categories-income')).toBeNull()
        expect(screen.getAllByRole('listitem')).toHaveLength(2)
    })
})

describe('CategoriesStep rows', () => {
    it('top-aligns the emoji and the remove button with the input', () => {
        render(<Harness />)
        const row = screen.getAllByRole('listitem')[0]
        expect(row.className).toContain('items-start')
        expect(row.className).not.toContain('items-center')
    })
})

describe('CategoriesStep descriptions', () => {
    const preset: DraftContent = {
        ...content,
        categories: [
            { name: '#UTILITIES', type: 'expense', icon: '💡', color: '#16a34a', isDefault: false },
            { name: 'Groceries', type: 'expense', icon: '🛒', color: '#16a34a', isDefault: false },
        ],
    }

    it('keeps the description out of the list, and gives only a preset category the info button', () => {
        render(<Harness initial={preset} />)
        expect(screen.getAllByRole('button', { name: 'spaces.create.categories.about' })).toHaveLength(1)
        const [presetRow, plainRow] = screen.getAllByRole('listitem')
        expect(within(presetRow).getByRole('button', { name: 'spaces.create.categories.about' })).toBeTruthy()
        expect(within(plainRow).queryByRole('button', { name: 'spaces.create.categories.about' })).toBeNull()
        expect(presetRow.querySelector('p')).toBeNull()
    })
})

describe('CategoriesStep add form', () => {
    it('adds a category of the chosen type (default: expense) and clears the field', () => {
        render(<Harness />)
        const input = screen.getByRole('textbox', { name: 'spaces.create.categories.new' })
        fireEvent.change(input, { target: { value: ' Fuel ' } })
        fireEvent.submit(screen.getByTestId('create-space-category-form'))
        const expenses = screen.getByTestId('create-space-categories-expense')
        expect(within(expenses).getByDisplayValue('Fuel')).toBeTruthy()
        expect((input as HTMLInputElement).value).toBe('')
    })

    it('shows the type icon in the closed select and hides the name on phones', () => {
        render(<Harness />)
        const trigger = screen.getByRole('combobox')
        // The value shown in the trigger is the item itself: icon, then the name that a phone hides.
        const name = trigger.querySelector('[data-type-name]')
        expect(name).not.toBeNull()
        expect(name?.previousElementSibling?.tagName.toLowerCase()).toBe('svg')
        expect(trigger.className).toContain('max-sm:[&_[data-type-name]]:hidden')
    })
})
