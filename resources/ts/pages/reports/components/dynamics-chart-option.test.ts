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

    it('tooltip skips zero values and fades rows other than the hovered series', () => {
        const option = dynamicsOption(input({ getHoveredIndex: () => 0 }))
        const html = option.tooltip.formatter([
            { seriesName: 'Food', seriesIndex: 0, value: 5, color: '#f00', axisValue: 'Jan' },
            { seriesName: 'Rent', seriesIndex: 1, value: 7, color: '#0f0', axisValue: 'Jan' },
            { seriesName: 'Fun', seriesIndex: 2, value: 0, color: '#00f', axisValue: 'Jan' },
        ])
        expect(html).toContain('Food')
        expect(html).toContain('Rent')
        expect(html).not.toContain('Fun')
        expect(html.match(/opacity:0\.4/g)).toHaveLength(1)
    })

    it('thins the day labels only when grouping by day', () => {
        expect(
            (dynamicsOption(input({ groupBy: 'day' })).xAxis.axisLabel as unknown as { interval: number }).interval,
        ).toBe(4)
        expect((dynamicsOption(input()).xAxis.axisLabel as unknown as { interval: number }).interval).toBe(0)
    })
})
