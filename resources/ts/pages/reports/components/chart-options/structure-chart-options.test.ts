import { describe, expect, it } from 'vitest'

import { barOption, chartLevel, donutOption, groupItems, isGrouped, treemapOption } from './structure-chart-options'

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

    it('gives a parent its parts in its shades, so the top level shows what is inside', () => {
        const [home, cinema] = chartLevel(items, null)
        expect(home.parts?.map((p) => [p.name, p.value])).toEqual([['Rent', 50], ['Utilities', 20]])
        expect(home.parts?.[0].color).toBe('#111111')
        expect(home.parts?.[1].color).not.toBe('#cccccc')
        expect(cinema.parts).toBeUndefined()
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

    it('keeps the total in the middle with the parts ring around', () => {
        const option = donutOption(input(chartLevel([{ name: 'Rent', value: 80, color: '#bbbbbb', parent: home }], null)))
        expect(option.series).toHaveLength(2)
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

describe('parts at the top level', () => {
    const theme = { tooltip: {}, surface: '#fff', text: '#000', textStrong: '#000' } as never
    const items = [
        { name: 'Cinema', value: 10, color: '#aaaaaa' },
        { name: 'Rent', value: 50, color: '#bbbbbb', parent: home },
        { name: 'Utilities', value: 20, color: '#cccccc', parent: home },
    ]
    const input = { items: chartLevel(items, null), total: 80, currency: 'USD', theme, chartWidth: 600 }
    type Datum = { name: string; value: number; itemStyle: { color: string } }

    it('rings the donut with the parts, lined up with their parent, legend naming parents only', () => {
        const option = donutOption(input)
        expect(option.series).toHaveLength(2)
        const outer = option.series[1].data as Datum[]
        // Home's slice is 70 wide: its parts fill exactly that, Cinema gets a blank gap.
        expect(outer.map((d) => d.value)).toEqual([50, 20, 10])
        expect(outer[2].itemStyle.color).toBe('transparent')
        expect(option.legend.data).toEqual(['Home', 'Cinema'])
    })

    it('draws the parts inside the main ring, and shrinks the total to fit the smaller hole', () => {
        const option = donutOption(input)
        const [main, parts] = option.series.map((s) => (s.radius as string[]).map(parseFloat))
        expect(parts[1]).toBeLessThan(main[0])
        const flat = donutOption({ ...input, items: chartLevel([items[0]], null) })
        const font = (o: typeof option) => (o.graphic[0] as { style: { fontSize: number } }).style.fontSize
        expect(font(option)).toBeLessThanOrEqual(font(flat))
    })

    it('keeps one ring when no parent has parts', () => {
        const flat = { ...input, items: chartLevel([items[0]], null) }
        expect(donutOption(flat).series).toHaveLength(1)
    })

    it('splits a parent’s bar into its parts, one stack per bar', () => {
        const option = barOption(input)
        const series = option.series as { stack?: string; data: (Datum | null)[] }[]
        // Bars are sorted largest first: Home, then Cinema.
        const home = series.map((s) => s.data[0]).filter((d): d is Datum => d !== null && d.value > 0)
        expect(home.map((d) => d.value)).toEqual([50, 20])
        const cinema = series.map((s) => s.data[1]).filter((d): d is Datum => d !== null && d.value > 0)
        expect(cinema.map((d) => d.value)).toEqual([10])
        expect(new Set(series.map((s) => s.stack))).toEqual(new Set(['total']))
    })
})
