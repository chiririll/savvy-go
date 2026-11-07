import ReactMarkdown, { type Components } from 'react-markdown'
import { useTranslation } from 'react-i18next'
import remarkGfm from 'remark-gfm'

import { cn } from '@/lib/utils'

import { safeUrl } from './safe-url'

// No headings and no images; raw HTML is never enabled (no rehype-raw), so it is dropped.
const DISALLOWED = ['img', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6']

// Task items are read-only; the checkbox has no label of its own, so give it a generic one.
function TaskCheckbox({ node: _node, ...props }: React.ComponentProps<'input'> & { node?: unknown }) {
    const { t } = useTranslation('forms')
    return <input {...props} aria-label={t('markdown.taskItem')} className="size-3.5 translate-y-0.5 accent-primary" />
}

const components: Components = {
    a: ({ href, children }) =>
        href ? (
            <a href={href} target="_blank" rel="noopener noreferrer" className="text-primary underline underline-offset-2">
                {children}
            </a>
        ) : (
            <span>{children}</span>
        ),
    ul: ({ node: _node, className, ...props }) => (
        <ul {...props} className={cn('list-disc pl-5', className?.includes('contains-task-list') && 'list-none pl-0', className)} />
    ),
    ol: ({ node: _node, ...props }) => <ol {...props} className="list-decimal pl-5" />,
    li: ({ node: _node, className, ...props }) => (
        <li {...props} className={cn(className?.includes('task-list-item') && 'flex items-baseline gap-2', className)} />
    ),
    input: TaskCheckbox,
    code: ({ node: _node, ...props }) => <code {...props} className="rounded bg-muted px-1 py-0.5 font-mono text-[0.85em]" />,
    pre: ({ node: _node, ...props }) => <pre {...props} className="overflow-x-auto rounded-md bg-muted p-2 text-xs" />,
    p: ({ node: _node, ...props }) => <p {...props} className="whitespace-pre-wrap" />,
}

export default function MarkdownContent({ children, className }: { children: string; className?: string }) {
    return (
        <div className={cn('flex flex-col gap-1 text-sm break-words', className)}>
            <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                disallowedElements={DISALLOWED}
                unwrapDisallowed
                urlTransform={safeUrl}
                components={components}
            >
                {children}
            </ReactMarkdown>
        </div>
    )
}
