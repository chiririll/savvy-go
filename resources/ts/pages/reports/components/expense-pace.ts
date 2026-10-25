import { formatExpensePaceMonthLabel } from '../utils'
import type { ExpensePaceMonth } from '@/api/reports'

export interface MonthChartData {
    label: string
    budget: number | null
    hasBudget: boolean
    idealPace: number[]
    actualExpenses: number[]
    currentDay: number | null
    currentActual: number
    budgetRemaining: number
    forecastTotal: number
    forecastDiff: number
    isOverBudget: boolean
    dailyAverage: number
    days: number[]
    daysInMonth: number
    totalSpent: number
}

/** The spend expected by month end: the pace so far for this month, the whole total for a finished one. */
function forecastOf(month: ExpensePaceMonth, currentActual: number) {
    const { currentDay, daysInMonth, totalSpent, dailyExpenses } = month
    if (currentDay !== null && currentDay > 0 && currentActual > 0) {
        const dailyAverage = currentActual / currentDay
        return { dailyAverage, forecastTotal: Math.round(dailyAverage * daysInMonth) }
    }
    const isPastMonth = currentDay === null && dailyExpenses.length > 0
    return isPastMonth
        ? { dailyAverage: totalSpent / daysInMonth, forecastTotal: totalSpent }
        : { dailyAverage: 0, forecastTotal: 0 }
}

export function processMonthData(month: ExpensePaceMonth): MonthChartData {
    const { budget, dailyExpenses, currentDay, daysInMonth, totalSpent, monthStart } = month
    const label = formatExpensePaceMonthLabel(monthStart)

    const hasBudget = budget !== null && budget > 0
    const isCurrentMonth = currentDay !== null && currentDay > 0

    const currentActual = isCurrentMonth ? (dailyExpenses[currentDay - 1] ?? 0) : totalSpent

    let idealPace: number[] = []
    let budgetRemaining = 0

    if (hasBudget && budget) {
        const dailyBudget = budget / daysInMonth
        idealPace = Array.from({ length: daysInMonth }, (_, i) => Math.round(dailyBudget * (i + 1)))
        budgetRemaining = budget - currentActual
    }

    const { forecastTotal, dailyAverage } = forecastOf(month, currentActual)

    const forecastDiff = hasBudget && budget ? forecastTotal - budget : 0
    const isOverBudget = forecastDiff > 0

    const days = Array.from({ length: daysInMonth }, (_, i) => i + 1)
    const actualExpenses = isCurrentMonth ? dailyExpenses.slice(0, currentDay) : dailyExpenses

    return {
        label,
        budget: hasBudget ? budget : null,
        hasBudget,
        idealPace,
        actualExpenses,
        currentDay: isCurrentMonth ? currentDay : null,
        currentActual,
        budgetRemaining,
        forecastTotal,
        forecastDiff,
        isOverBudget,
        dailyAverage,
        days,
        daysInMonth,
        totalSpent,
    }
}
