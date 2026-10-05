import { useMemo } from 'react'
import { useTheme } from '@/hooks/use-theme'

/** Neutral chart colors, matched to the app's light/dark design tokens (see index.css). */
export interface ChartTheme {
    isDark: boolean
    text: string
    textStrong: string
    axisLine: string
    splitLine: string
    /** Card background: separates slices/markers from the surface behind them. */
    surface: string
    tooltip: { backgroundColor: string; borderColor: string; textStyle: { color: string } }
}

const LIGHT: ChartTheme = {
    isDark: false,
    text: '#737373',
    textStrong: '#171717',
    axisLine: '#e5e5e5',
    splitLine: '#f0f0f0',
    surface: '#ffffff',
    tooltip: { backgroundColor: '#ffffff', borderColor: '#e5e5e5', textStyle: { color: '#171717' } },
}

const DARK: ChartTheme = {
    isDark: true,
    text: '#a3a3a3',
    textStrong: '#fafafa',
    axisLine: '#404040',
    splitLine: '#2a2a2a',
    surface: '#171717',
    tooltip: { backgroundColor: '#262626', borderColor: '#404040', textStyle: { color: '#fafafa' } },
}

/** Semantic series colors, readable on both themes. */
export const CHART_COLORS = {
    income: '#22c55e',
    expense: '#ef4444',
    balance: '#3b82f6',
    neutral: '#94a3b8',
} as const

export function useChartTheme(): ChartTheme {
    const { theme } = useTheme()
    return theme === 'dark' ? DARK : LIGHT
}

/** `#rrggbb` → `rgba(r, g, b, alpha)`. */
export function withAlpha(hex: string, alpha: number): string {
    const n = parseInt(hex.slice(1), 16)
    return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`
}

/** Top-to-bottom fade for area fills. */
export function verticalFade(hex: string, from: number, to = 0) {
    return {
        type: 'linear',
        x: 0,
        y: 0,
        x2: 0,
        y2: 1,
        colorStops: [
            { offset: 0, color: withAlpha(hex, from) },
            { offset: 1, color: withAlpha(hex, to) },
        ],
    }
}

export function legendTextStyle(c: ChartTheme, fontSize = 12) {
    return { fontSize, color: c.text }
}

/** Shared axis styling; pass `extra` to override or extend a section. */
export function axisStyle(c: ChartTheme, kind: 'category' | 'value', extra: Record<string, unknown> = {}) {
    const { axisLabel, ...rest } = extra as { axisLabel?: Record<string, unknown> }
    return kind === 'category'
        ? {
            type: 'category',
            axisLine: { lineStyle: { color: c.axisLine } },
            axisTick: { show: false },
            axisLabel: { fontSize: 11, color: c.text, rotate: 45, ...axisLabel },
            ...rest,
        }
        : {
            type: 'value',
            splitLine: { lineStyle: { color: c.splitLine, type: 'dashed' } },
            axisLabel: { fontSize: 11, color: c.text, ...axisLabel },
            ...rest,
        }
}
