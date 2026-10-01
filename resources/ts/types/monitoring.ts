export interface VolumeStats {
    disk: string
    path: string
    total_bytes: number | null
    free_bytes: number | null
    used_bytes: number | null
    used_percent: number | null
}

export interface BucketUsage {
    bucket: string
    bytes: number
    objects: number
}

export interface ManagedStorage {
    used_bytes: number
    objects: number
    pending_bytes: number
    buckets: BucketUsage[]
}

export interface UploadsStats {
    total: number
    by_status: Record<string, number>
}

export interface ImportsStats {
    total: number
    by_status: Record<string, number>
}

export interface StorageSnapshot {
    volume: VolumeStats
    managed: ManagedStorage
    uploads: UploadsStats
    imports: ImportsStats
}

export interface CpuStats {
    cores: number | null
    load: number[] | null
    load_percent: number | null
}

export interface MemoryStats {
    total_bytes: number | null
    used_bytes: number | null
    free_bytes: number | null
    used_percent: number | null
    source: 'cgroup' | 'host' | null
}

export interface ProcessStats {
    memory_bytes: number
    peak_bytes: number
    limit_bytes: number | null
    goroutines: number
    gc_cycles: number
}

export interface RuntimeStats {
    go_version: string
    app_version: string
    environment: string
    uptime_seconds: number | null
}

export interface ResourceSnapshot {
    cpu: CpuStats
    memory: MemoryStats
    process: ProcessStats
    runtime: RuntimeStats
}
