import { useTranslation } from 'react-i18next'
import { LineChart, BarChart3, ChevronDown } from 'lucide-react'
import { testIdControl, testIdControls } from '@/lib/test-id'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import type { DynamicsChartType } from './dynamics-chart-option'

export interface SeriesConfig {
    id: number
    name: string
    color: string
    enabled: boolean
}

interface ChartTypeBadgesProps {
    value: DynamicsChartType
    onChange: (type: DynamicsChartType) => void
}

export function ChartTypeBadges({ value, onChange }: ChartTypeBadgesProps) {
    const { t } = useTranslation('pages')
    const options = [
        { type: 'line', Icon: LineChart, label: t('reports.views.line') },
        { type: 'bar', Icon: BarChart3, label: t('reports.views.bar') },
    ] as const

    return (
        <div className="flex gap-1" {...testIdControls('chart-type')}>
            {options.map(({ type, Icon, label }) => (
                <Badge
                    key={type}
                    {...testIdControl(type, value === type)}
                    variant={value === type ? 'default' : 'outline'}
                    className="cursor-pointer gap-1.5"
                    onClick={() => onChange(type)}
                >
                    <Icon className="size-3.5" />
                    {label}
                </Badge>
            ))}
        </div>
    )
}

interface SeriesPickerProps {
    label: string
    series: SeriesConfig[]
    onToggle: (id: number) => void
}

export function SeriesPicker({ label, series, onToggle }: SeriesPickerProps) {
    const enabledCount = series.filter((item) => item.enabled).length

    return (
        <Popover>
            <PopoverTrigger asChild>
                <Button variant="outline" size="sm" className="h-7 gap-1">
                    {label}
                    <Badge variant="secondary" className="ml-1 px-1.5 text-xs">
                        {enabledCount}
                    </Badge>
                    <ChevronDown className="size-3" />
                </Button>
            </PopoverTrigger>
            <PopoverContent className="w-56 p-2" align="end">
                <div className="space-y-1">
                    {series.map((item) => (
                        <div
                            key={item.id}
                            className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-muted cursor-pointer"
                            onClick={() => onToggle(item.id)}
                        >
                            <Checkbox checked={item.enabled} className="pointer-events-none" />
                            <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: item.color }} />
                            <Label className="text-sm cursor-pointer flex-1">{item.name}</Label>
                        </div>
                    ))}
                </div>
            </PopoverContent>
        </Popover>
    )
}
