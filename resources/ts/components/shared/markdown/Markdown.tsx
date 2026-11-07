import { lazy, Suspense } from 'react'

import { cn } from '@/lib/utils'

// Kept out of the entry chunk: the parser is only needed where a note is shown.
const MarkdownContent = lazy(() => import('./MarkdownContent'))

export function Markdown({ children, className }: { children: string; className?: string }) {
    return (
        <Suspense fallback={<div className={cn('text-sm break-words whitespace-pre-wrap', className)}>{children}</div>}>
            <MarkdownContent className={className}>{children}</MarkdownContent>
        </Suspense>
    )
}
