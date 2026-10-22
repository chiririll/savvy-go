export type Severity = 'ok' | 'warn' | 'crit'

export function severityFor(percent: number | null): Severity {
    if (percent === null) return 'ok'
    if (percent >= 90) return 'crit'
    if (percent >= 75) return 'warn'
    return 'ok'
}

export const SEVERITY_STROKE: Record<Severity, string> = {
    ok: 'text-emerald-500',
    warn: 'text-amber-500',
    crit: 'text-red-500',
}

export const SEVERITY_FILL: Record<Severity, string> = {
    ok: 'bg-emerald-500',
    warn: 'bg-amber-500',
    crit: 'bg-red-500',
}
