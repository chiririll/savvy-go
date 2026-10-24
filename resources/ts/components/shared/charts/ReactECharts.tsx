import * as echarts from 'echarts/core'
import { BarChart, LineChart, PieChart, SankeyChart, ScatterChart, TreemapChart } from 'echarts/charts'
import {
    AriaComponent,
    AxisPointerComponent,
    GraphicComponent,
    GridComponent,
    LegendComponent,
    MarkLineComponent,
    TooltipComponent,
} from 'echarts/components'
import { SVGRenderer } from 'echarts/renderers'
import ReactEChartsCore from 'echarts-for-react/lib/core'
import type { EChartsReactProps } from 'echarts-for-react/lib/types'

echarts.use([
    BarChart,
    LineChart,
    PieChart,
    SankeyChart,
    ScatterChart,
    TreemapChart,
    AriaComponent,
    AxisPointerComponent,
    GraphicComponent,
    GridComponent,
    LegendComponent,
    MarkLineComponent,
    TooltipComponent,
    SVGRenderer,
])

const prefersReducedMotion = () =>
    typeof window !== 'undefined' && window.matchMedia?.('(prefers-reduced-motion: reduce)').matches

/**
 * ECharts with only the charts/components this app uses registered (tree-shaken).
 * Charts draw without animation when the user asks for reduced motion.
 * `ariaLabel` names the chart for screen readers (role="img" on the container).
 */
export default function ReactECharts({
    option,
    opts,
    ariaLabel,
    ...props
}: Omit<EChartsReactProps, 'echarts'> & { ariaLabel?: string }) {
    const named = ariaLabel ? { ...option, aria: { enabled: true, label: { description: ariaLabel } } } : option
    return (
        <ReactEChartsCore
            echarts={echarts}
            option={prefersReducedMotion() ? { ...named, animation: false } : named}
            opts={{ renderer: 'svg', ...opts }}
            {...props}
        />
    )
}
