import { describe, expect, it } from 'vitest'

import { chartLevel, donutOption, groupItems, isGrouped, treemapOption } from './structure-chart-options'

const home = { name: 'Home', color: '#111111' }

describe('groupItems', () => {
    const items = [
        { name: 'Cinema', value: 10, color: '#aaaaaa' },
        { name: 'Rent', value: 50, color: '#bbbbbb', parent: home },
        { name: 'Home', value: 5, color: '#111111', parent: home },
        { name: 'Utilities', value: 20, color: '#cccccc', parent: home },
    ]

    it('rolls children and the parent’s own lines into the parent, largest first', () => {
        const groups = groupItems(items)
        expect(groups.map((g) => [g.name, g.value])).toEqual([['Home', 75], ['Cinema', 10]])
        expect(groups[0].children.map((c) => c.name)).toEqual(['Rent', 'Utilities', 'Home'])
        expect(groups[0].color).toBe('#111111')
    })

    it('keeps a category without a parent as a group of its own', () => {
        expect(groupItems([items[0]])).toHaveLength(1)
        expect(isGrouped([items[0]])).toBe(false)
        expect(isGrouped(items)).toBe(true)
    })
})

describe('chartLevel', () => {
    const items = [
        { name: 'Cinema', value: 10, color: '#aaaaaa' },
        { name: 'Rent', value: 50, color: '#bbbbbb', parent: home },
        { name: 'Utilities', value: 20, color: '#cccccc', parent: home },
    ]

    it('shows one entry per parent at the top, and only parents open', () => {
        const level = chartLevel(items, null)
        expect(level.map((i) => [i.name, i.value, i.drill])).toEqual([['Home', 70, 'Home'], ['Cinema', 10, undefined]])
        expect(level[0].color).toBe('#111111')
    })

    it('opens a parent into its children, coloured in its shades', () => {
        const level = chartLevel(items, 'Home')
        expect(level.map((i) => [i.name, i.value])).toEqual([['Rent', 50], ['Utilities', 20]])
        expect(level[0].color).toBe('#111111')
        expect(level[1].color).not.toBe('#cccccc')
        expect(level.every((i) => i.drill === undefined)).toBe(true)
    })

    it('falls back to the top when the opened parent is gone', () => {
        expect(chartLevel(items, 'Nope').map((i) => i.name)).toEqual(['Home', 'Cinema'])
    })
})

describe('options', () => {
    const theme = { tooltip: {}, surface: '#fff', text: '#000', textStrong: '#000' } as never
    const input = (items: Parameters<typeof chartLevel>[0]) => ({ items, total: 80, currency: 'USD', theme, chartWidth: 600 })

    it('draws one ring with the total in the middle', () => {
        const option = donutOption(input(chartLevel([{ name: 'Rent', value: 80, color: '#bbbbbb', parent: home }], null)))
        expect(option.series).toHaveLength(1)
        expect(option.graphic).toHaveLength(2)
    })

    it('nests a treemap group and shades its tiles', () => {
        const option = treemapOption(input([
            { name: 'Rent', value: 50, color: '#bbbbbb', parent: home },
            { name: 'Utilities', value: 30, color: '#cccccc', parent: home },
        ]))
        const [group] = option.series[0].data as { children: { itemStyle: { color: string } }[] }[]
        expect(group.children.map((c) => c.itemStyle.color)[0]).toBe('#111111')
    })
})
