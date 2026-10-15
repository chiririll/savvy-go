import type { BootstrapHint } from '@/types'

/** Commands that show the bootstrap password; none for a plain binary, whose terminal and file are enough. */
export function bootstrapCommands(hint: BootstrapHint): string[] {
    switch (hint.environment) {
        case 'docker':
            return ['docker logs <container>', `docker exec <container> cat ${hint.file}`]
        case 'systemd':
            return ['journalctl -u savvy-go', `sudo cat ${hint.file}`]
        case 'binary':
            return []
    }
}
