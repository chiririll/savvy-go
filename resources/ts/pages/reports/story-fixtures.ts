import { expect, waitFor } from 'storybook/test'

const DRAWN = '[_echarts_instance_] svg path, [_echarts_instance_] svg rect'

/** Waits until a chart has drawn something: ECharts marks its host and fills it with SVG shapes. */
export async function expectChartDrawn(root: HTMLElement) {
    await waitFor(() => expect(root.querySelector(DRAWN)).not.toBeNull())
}

/** The text drawn inside the charts of `root` (labels of nodes, axes and legends). */
export const chartText = (root: HTMLElement) => root.querySelector('[_echarts_instance_]')?.textContent ?? ''
