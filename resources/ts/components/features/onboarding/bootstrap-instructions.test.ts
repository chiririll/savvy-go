import { describe, expect, it } from 'vitest'
import { bootstrapCommands } from './bootstrap-instructions'

describe('bootstrapCommands', () => {
    it('reads the log and the file of a docker container', () => {
        expect(bootstrapCommands({ environment: 'docker', file: '/data/bootstrap-password' })).toEqual([
            'docker logs <container>',
            'docker exec <container> cat /data/bootstrap-password',
        ])
    })

    it('reads the journal and the file of a systemd service', () => {
        expect(bootstrapCommands({ environment: 'systemd', file: '/var/lib/savvy-go/bootstrap-password' })).toEqual([
            'journalctl -u savvy-go',
            'sudo cat /var/lib/savvy-go/bootstrap-password',
        ])
    })

    it('gives a plain binary no command: the terminal and the file are enough', () => {
        expect(bootstrapCommands({ environment: 'binary', file: './savvy-data/bootstrap-password' })).toEqual([])
    })
})
