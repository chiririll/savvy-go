export type Severity = 'ok' | 'warn' | 'crit'

export function severityFor(percent: number | null): Severity {
    if (percent === null) return 'ok'
    if (percent >= 90) return 'crit'
    if (percent >= 75) return 'warn'
    return 'ok'
}

export const SEVERITY_STROKE: Record<Severity, string> = {
    ok: 'text-success',
    warn: 'text-warning',
    crit: 'text-destructive',
}

export const SEVERITY_FILL: Record<Severity, string> = {
    ok: 'bg-success',
    warn: 'bg-warning',
    crit: 'bg-destructive',
}
