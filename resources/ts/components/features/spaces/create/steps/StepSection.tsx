import type { ReactNode } from 'react'

/** A titled block of a wizard step. */
export function StepSection({ title, hint, children }: { title: string; hint?: string; children: ReactNode }) {
    return (
        <section className="space-y-2">
            <div>
                <h3 className="text-sm font-semibold">{title}</h3>
                {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
            </div>
            {children}
        </section>
    )
}
