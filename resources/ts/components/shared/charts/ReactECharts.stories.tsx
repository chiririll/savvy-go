import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, waitFor } from 'storybook/test'

import ReactECharts from './ReactECharts'

const meta = {
    title: 'shared/charts/ReactECharts',
    component: ReactECharts,
    // The chart takes the height it is given.
    args: { style: { height: 240, width: '100%' } },
} satisfies Meta<typeof ReactECharts>

export default meta
type Story = StoryObj<typeof meta>

const months = ['Jan', 'Feb', 'Mar', 'Apr']

export const Bars: Story = {
    args: {
        option: {
            xAxis: { type: 'category', data: months },
            yAxis: { type: 'value' },
            series: [{ type: 'bar', data: [120, 200, 150, 80] }],
        },
    },
    // The SVG renderer is the one registered: no canvas.
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(canvasElement.querySelector('svg')).not.toBeNull())
        await expect(canvasElement.querySelector('canvas')).toBeNull()
    },
}

export const Line: Story = {
    args: {
        option: {
            xAxis: { type: 'category', data: months },
            yAxis: { type: 'value' },
            series: [{ type: 'line', data: [5, 9, 7, 12], smooth: true }],
            tooltip: { trigger: 'axis' },
        },
    },
}

export const Pie: Story = {
    args: {
        option: {
            series: [
                {
                    type: 'pie',
                    radius: ['40%', '70%'],
                    data: [
                        { name: 'Food', value: 40 },
                        { name: 'Rent', value: 55 },
                        { name: 'Fun', value: 15 },
                    ],
                },
            ],
        },
    },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(canvasElement.querySelector('svg path')).not.toBeNull())
    },
}

export const NoData: Story = {
    args: {
        option: { xAxis: { type: 'category', data: [] }, yAxis: { type: 'value' }, series: [{ type: 'bar', data: [] }] },
    },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(canvasElement.querySelector('svg')).not.toBeNull())
    },
}
