import { useTranslation } from 'react-i18next'
import type { BootstrapHint } from '@/types'
import { bootstrapCommands } from './bootstrap-instructions'

export function BootstrapInstructions({ hint }: { hint: BootstrapHint }) {
    const { t } = useTranslation('auth')
    const commands = bootstrapCommands(hint)

    return (
        <div className="space-y-2 rounded-md border bg-muted/40 p-3 text-sm text-muted-foreground">
            {/* i18n-dynamic: one text per environment (docker, systemd, binary) */}
            <p>{t(`setup.where.${hint.environment}`, { file: hint.file })}</p>
            {commands.map((command) => (
                <code
                    key={command}
                    className="block overflow-x-auto rounded bg-muted px-2 py-1 font-mono text-xs text-foreground"
                >
                    {command}
                </code>
            ))}
        </div>
    )
}
