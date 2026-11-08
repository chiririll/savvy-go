// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { CategoryExpense } from '@/api/reports'
import { DEFAULT_FILTERS } from '../../types'
import { ExpensesByCategory } from './ExpensesByCategory'

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({
        t: (key: string, opts?: { name?: string }) => (opts?.name ? `${key}:${opts.name}` : key),
        i18n: { language: 'en' },
    }),
}))

const navigate = vi.fn()
vi.mock('react-router-dom', () => ({ useNavigate: () => navigate }))

const row = (id: number, name: string, current: number, extra: Partial<CategoryExpense> = {}): CategoryExpense => ({
    id, name, icon: '🎬', color: '#2563eb', current, previous: 0, ...extra,
})
const categories = [
    row(1, 'Leisure', 45, { children: [row(2, 'Cinema', 40, { parentId: 1 }), row(1, 'Leisure', 5, { parentId: 1 })] }),
    row(3, 'Food', 30),
]
vi.mock('@/hooks', () => ({
    useExpensesByCategory: () => ({ data: { categories, currency: 'USD' }, isLoading: false, error: null }),
}))

afterEach(() => {
    cleanup()
    navigate.mockReset()
})

describe('ExpensesByCategory', () => {
    it('lists top-level categories and keeps subcategories folded', () => {
        render(<ExpensesByCategory filters={DEFAULT_FILTERS} />)
        expect(screen.getByText('Leisure')).toBeTruthy()
        expect(screen.getByText('Food')).toBeTruthy()
        expect(screen.queryByText('Cinema')).toBeNull()
    })

    it('opens a parent into its subcategories, naming its own lines apart', () => {
        render(<ExpensesByCategory filters={DEFAULT_FILTERS} />)
        const toggle = screen.getByRole('button', { name: 'reports.structure.expand:Leisure' })
        fireEvent.click(toggle)
        expect(toggle.getAttribute('aria-expanded')).toBe('true')
        expect(screen.getByText('Cinema')).toBeTruthy()
        expect(screen.getByText('reports.structure.ownLines:Leisure')).toBeTruthy()
    })

    it('opens a parent’s transactions with its subcategories, a child’s alone', () => {
        render(<ExpensesByCategory filters={DEFAULT_FILTERS} />)
        fireEvent.click(screen.getByText('Leisure'))
        expect(navigate).toHaveBeenLastCalledWith('/transactions?category_ids=1,2')
        fireEvent.click(screen.getByRole('button', { name: 'reports.structure.expand:Leisure' }))
        fireEvent.click(screen.getByText('Cinema'))
        expect(navigate).toHaveBeenLastCalledWith('/transactions?category_ids=2')
    })
})
