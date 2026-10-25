import { cn } from '@/lib/utils'

import { SEVERITY_STROKE, severityFor } from './severity'

interface GaugeProps {
    percent: number | null
    label?: string
    size?: number
}

export function Gauge({ percent, label = 'Used', size = 196 }: GaugeProps) {
    const severity = severityFor(percent)
    const stroke = size >= 180 ? 14 : 11
    const radius = (size - stroke) / 2
    const circumference = 2 * Math.PI * radius
    const clamped = Math.max(0, Math.min(100, percent ?? 0))
    const offset = circumference - (clamped / 100) * circumference
    const ring = { cx: size / 2, cy: size / 2, r: radius, fill: 'none', strokeWidth: stroke }

    return (
        <div className="relative" style={{ width: size, height: size }}>
            <svg width={size} height={size} className="-rotate-90">
                <circle {...ring} className="stroke-muted" />
                <circle
                    {...ring}
                    strokeLinecap="round"
                    strokeDasharray={circumference}
                    strokeDashoffset={offset}
                    className={cn('transition-[stroke-dashoffset] duration-700 ease-out', SEVERITY_STROKE[severity])}
                />
            </svg>
            <div className="absolute inset-0 flex flex-col items-center justify-center">
                <span className={cn('font-bold tabular-nums tracking-tight', size >= 180 ? 'text-4xl' : 'text-3xl')}>
                    {percent === null ? '—' : `${percent}%`}
                </span>
                <span className="mt-1 text-xs font-medium uppercase tracking-wider text-muted-foreground">{label}</span>
            </div>
        </div>
    )
}
