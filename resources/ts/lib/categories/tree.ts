import type { Category } from '@/types'

export interface CategoryTreeNode {
    category: Category
    /** Set for a subcategory whose parent is in the list. */
    parent: Category | null
    childrenCount: number
}

/**
 * Orders categories as a tree one level deep: each top-level category in the
 * given order, followed by its subcategories in the given order. A
 * subcategory whose parent is missing from the list stays at the top level.
 */
export function categoryTree(categories: Category[]): CategoryTreeNode[] {
    const byId = new Map(categories.map((c) => [c.id, c]))
    const children = new Map<number, Category[]>()
    const roots: Category[] = []

    for (const c of categories) {
        const parent = c.parentId != null ? byId.get(c.parentId) : undefined
        if (parent) {
            children.set(parent.id, [...(children.get(parent.id) ?? []), c])
        } else {
            roots.push(c)
        }
    }

    return roots.flatMap((root) => {
        const own = children.get(root.id) ?? []
        return [
            { category: root, parent: null, childrenCount: own.length },
            ...own.map((child) => ({ category: child, parent: root, childrenCount: 0 })),
        ]
    })
}
