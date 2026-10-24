// Sample monitoring snapshots for the stories of this folder.
import type { ResourceSnapshot, StorageSnapshot } from '@/types/monitoring'

const GB = 1024 ** 3
const MB = 1024 ** 2

export const resources = (overrides: Partial<ResourceSnapshot> = {}): ResourceSnapshot => ({
    cpu: { cores: 4, load: [0.42, 0.61, 0.55], load_percent: 12 },
    memory: { total_bytes: 2 * GB, used_bytes: 1.2 * GB, free_bytes: 0.8 * GB, used_percent: 60, source: 'cgroup' },
    process: { memory_bytes: 96 * MB, peak_bytes: 140 * MB, limit_bytes: 512 * MB, goroutines: 1234, gc_cycles: 87 },
    runtime: { go_version: 'go1.25.1', app_version: '1.4.0', environment: 'production', uptime_seconds: 3 * 86400 + 5 * 3600 },
    ...overrides,
})

export const storage = (overrides: Partial<StorageSnapshot> = {}): StorageSnapshot => ({
    volume: {
        disk: 'local',
        path: '/var/lib/savvy/uploads',
        total_bytes: 100 * GB,
        free_bytes: 60 * GB,
        used_bytes: 40 * GB,
        used_percent: 40,
    },
    managed: {
        used_bytes: 3 * GB,
        objects: 4210,
        pending_bytes: 12 * MB,
        buckets: [
            { bucket: 'backups', bytes: 2.5 * GB, objects: 30 },
            { bucket: 'transaction-imports', bytes: 0.5 * GB, objects: 4180 },
        ],
    },
    uploads: { total: 4300, by_status: { pending: 2, completed: 4298 } },
    imports: { total: 18, by_status: { completed: 17, failed: 1 } },
    ...overrides,
})

export { GB, MB }
