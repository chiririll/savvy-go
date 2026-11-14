import { Bold, Code, Italic, Link, List, ListChecks, X, type LucideIcon } from 'lucide-react'
import { useId, useLayoutEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { useAutoGrow } from '@/hooks/use-auto-grow'
import {
    continueList,
    insertLink,
    toggleBold,
    toggleCode,
    toggleItalic,
    toggleList,
    type MarkdownEdit,
} from '@/lib/ui/markdown-edit'
import { cn } from '@/lib/utils'

import { Markdown } from './Markdown'

interface MarkdownFieldProps {
    value: string
    onChange: (value: string) => void
    onBlur?: () => void
    id?: string
    name?: string
    placeholder?: string
    disabled?: boolean
    'aria-invalid'?: boolean
    className?: string
    /** Visible rows of the editor, default 4. Compact fields use 2. */
    rows?: number
    /** Shows a cross at the right end of the toolbar that calls this, e.g. to fold the editor into a compact field. */
    onClose?: () => void
}

type Action = 'bold' | 'italic' | 'bulletList' | 'taskList' | 'link' | 'code'

const ACTIONS: { action: Action; icon: LucideIcon; run: (e: MarkdownEdit) => MarkdownEdit }[] = [
    { action: 'bold', icon: Bold, run: toggleBold },
    { action: 'italic', icon: Italic, run: toggleItalic },
    { action: 'bulletList', icon: List, run: (e) => toggleList(e, 'bullet') },
    { action: 'taskList', icon: ListChecks, run: (e) => toggleList(e, 'task') },
    { action: 'link', icon: Link, run: insertLink },
    { action: 'code', icon: Code, run: toggleCode },
]

const SHORTCUTS: Record<string, Action> = { b: 'bold', i: 'italic', k: 'link' }

export function MarkdownField({
    value,
    onChange,
    onBlur,
    id,
    name,
    placeholder,
    disabled,
    className,
    rows = 4,
    onClose,
    ...rest
}: MarkdownFieldProps) {
    const { t } = useTranslation('forms')
    const [mode, setMode] = useState<'write' | 'preview'>('write')
    const ref = useRef<HTMLTextAreaElement>(null)
    const pending = useRef<{ start: number; end: number } | null>(null)
    const autoId = useId()
    const fieldId = id ?? autoId
    const writing = mode === 'write'
    // The editor grows with the text, up to its cap; back from the preview it is measured again.
    useAutoGrow(ref, [value, writing])

    // Restore the selection after React has written the new text into the textarea.
    useLayoutEffect(() => {
        const sel = pending.current
        if (!sel || !ref.current) return
        pending.current = null
        ref.current.focus()
        ref.current.setSelectionRange(sel.start, sel.end)
    }, [value])

    const apply = (edit: MarkdownEdit) => {
        pending.current = { start: edit.start, end: edit.end }
        if (edit.text === value) {
            // No text change means no re-render; set the selection now.
            const sel = pending.current
            pending.current = null
            ref.current?.setSelectionRange(sel.start, sel.end)
            return
        }
        onChange(edit.text)
    }

    const run = (action: Action) => {
        const el = ref.current
        if (!el || disabled) return
        const entry = ACTIONS.find((a) => a.action === action)
        if (!entry) return
        apply(entry.run({ text: el.value, start: el.selectionStart, end: el.selectionEnd }))
    }

    const onKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
        if ((e.ctrlKey || e.metaKey) && !e.altKey && !e.shiftKey) {
            const action = SHORTCUTS[e.key.toLowerCase()]
            if (action) {
                e.preventDefault()
                run(action)
            }
            return
        }
        if (e.key === 'Enter' && !e.shiftKey && !e.ctrlKey && !e.metaKey && !e.altKey && !e.nativeEvent.isComposing) {
            const el = e.currentTarget
            const edit = continueList({ text: el.value, start: el.selectionStart, end: el.selectionEnd })
            if (edit) {
                e.preventDefault()
                apply(edit)
            }
        }
    }

    const labels: Record<Action, string> = {
        bold: t('markdown.bold'),
        italic: t('markdown.italic'),
        bulletList: t('markdown.bulletList'),
        taskList: t('markdown.taskList'),
        link: t('markdown.link'),
        code: t('markdown.code'),
    }

    return (
        <div className={cn('flex flex-col gap-1.5', className)}>
            <div className="flex items-center justify-between gap-2">
                <div role="toolbar" aria-label={t('markdown.toolbar')} className="flex min-w-0 items-center gap-0.5 overflow-x-auto">
                    {ACTIONS.map(({ action, icon: Icon }) => (
                        <Button
                            key={action}
                            type="button"
                            variant="ghost"
                            size="icon-sm"
                            aria-label={labels[action]}
                            title={labels[action]}
                            disabled={disabled || !writing}
                            // Keep the caret in the textarea when a button is pressed.
                            onMouseDown={(e) => e.preventDefault()}
                            onClick={() => run(action)}
                        >
                            <Icon />
                        </Button>
                    ))}
                </div>
                <div className="flex shrink-0 rounded-md bg-muted p-0.5" role="group">
                    {(['write', 'preview'] as const).map((m) => (
                        <Button
                            key={m}
                            type="button"
                            variant={mode === m ? 'secondary' : 'ghost'}
                            size="sm"
                            className="h-7 px-2"
                            aria-pressed={mode === m}
                            onClick={() => setMode(m)}
                        >
                            {m === 'write' ? t('markdown.write') : t('markdown.preview')}
                        </Button>
                    ))}
                    {onClose && (
                        <Button
                            type="button"
                            variant="ghost"
                            size="icon-sm"
                            aria-label={t('markdown.close')}
                            title={t('markdown.close')}
                            className="ml-1"
                            onClick={onClose}
                        >
                            <X />
                        </Button>
                    )}
                </div>
            </div>
            {writing ? (
                <Textarea
                    ref={ref}
                    id={fieldId}
                    name={name}
                    value={value}
                    rows={rows}
                    placeholder={placeholder}
                    disabled={disabled}
                    aria-invalid={rest['aria-invalid']}
                    onChange={(e) => onChange(e.target.value)}
                    onBlur={onBlur}
                    onKeyDown={onKeyDown}
                    className="field-sizing-fixed max-h-72 overflow-y-auto font-mono text-sm md:text-sm"
                />
            ) : (
                <div className="min-h-16 rounded-md border bg-transparent px-3 py-2">
                    {value.trim() === '' ? (
                        <p className="text-sm text-muted-foreground">{t('markdown.previewEmpty')}</p>
                    ) : (
                        <Markdown>{value}</Markdown>
                    )}
                </div>
            )}
        </div>
    )
}
