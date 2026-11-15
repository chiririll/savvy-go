import { describe, expect, it } from 'vitest'

import { category } from '@/test/fixtures'
import { categoryTree } from './tree'

const rows = (tree: ReturnType<typeof categoryTree>) =>
    tree.map((node) => [node.category.name, node.parent?.name ?? null, node.childrenCount])

describe('categoryTree', () => {
    it('puts each subcategory right after its parent, keeping the given order', () => {
        const tree = categoryTree([
            category({ id: 1, name: 'Car' }),
            category({ id: 2, name: 'Fuel', parentId: 1 }),
            category({ id: 3, name: 'Groceries' }),
            category({ id: 4, name: 'Home' }),
            category({ id: 5, name: 'Parking', parentId: 1 }),
            category({ id: 6, name: 'Rent', parentId: 4 }),
        ])

        expect(rows(tree)).toEqual([
            ['Car', null, 2],
            ['Fuel', 'Car', 0],
            ['Parking', 'Car', 0],
            ['Groceries', null, 0],
            ['Home', null, 1],
            ['Rent', 'Home', 0],
        ])
    })

    it('shows a subcategory at the top level when its parent is not in the list', () => {
        const tree = categoryTree([category({ id: 2, name: 'Fuel', parentId: 1 }), category({ id: 3, name: 'Groceries' })])

        expect(rows(tree)).toEqual([
            ['Fuel', null, 0],
            ['Groceries', null, 0],
        ])
    })

    it('lists a subcategory placed before its parent under that parent', () => {
        const tree = categoryTree([category({ id: 2, name: 'Alpha', parentId: 1 }), category({ id: 1, name: 'Zulu' })])

        expect(rows(tree)).toEqual([
            ['Zulu', null, 1],
            ['Alpha', 'Zulu', 0],
        ])
    })
})
