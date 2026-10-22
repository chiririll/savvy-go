import * as echarts from 'echarts/core'
import { BarChart, LineChart, PieChart, SankeyChart, ScatterChart, TreemapChart } from 'echarts/charts'
import {
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
 */
export default function ReactECharts({ option, opts, ...props }: Omit<EChartsReactProps, 'echarts'>) {
    return (
        <ReactEChartsCore
            echarts={echarts}
            option={prefersReducedMotion() ? { ...option, animation: false } : option}
            opts={{ renderer: 'svg', ...opts }}
            {...props}
        />
    )
}
