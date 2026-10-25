import { describe, expect, it } from 'vitest'

import type { ChartTheme } from '@/lib/chart-theme'

import { dynamicsOption, type DynamicsOptionInput } from './dynamics-chart-option'

const theme = { tooltip: {}, text: '#000', textStrong: '#000', surface: '#fff' } as unknown as ChartTheme

const input = (overrides: Partial<DynamicsOptionInput> = {}): DynamicsOptionInput => ({
    labels: ['Jan', 'Feb'],
    datasets: [{ name: 'Food', color: '#ff0000', data: [1, 2] }],
    chartType: 'line',
    groupBy: 'month',
    currency: 'USD',
    theme,
    isNarrow: false,
    ...overrides,
})

describe('dynamicsOption', () => {
    it('fills the area only under a lone line', () => {
        const [lone] = dynamicsOption(input()).series
        expect(lone.areaStyle).toBeDefined()

        const many = dynamicsOption(
            input({
                datasets: [
                    { name: 'A', color: '#ff0000', data: [1] },
                    { name: 'B', color: '#00ff00', data: [2] },
                ],
            }),
        )
        expect(many.series.every((item) => item.areaStyle === undefined)).toBe(true)
        expect(many.legend).toBeDefined()
    })

    it('rounds bar tops and drops line styling for bars', () => {
        const [bar] = dynamicsOption(input({ chartType: 'bar' })).series
        expect(bar.itemStyle.borderRadius).toEqual([4, 4, 0, 0])
        expect(bar.lineStyle).toBeUndefined()
    })

    it('thins the day labels only when grouping by day', () => {
        expect(
            (dynamicsOption(input({ groupBy: 'day' })).xAxis.axisLabel as unknown as { interval: number }).interval,
        ).toBe(4)
        expect((dynamicsOption(input()).xAxis.axisLabel as unknown as { interval: number }).interval).toBe(0)
    })
})
