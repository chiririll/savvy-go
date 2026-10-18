// What the demo seed writes to the manifest of --seed-config; see internal/seed/manifest.go.
export interface ManifestSpace {
    id: number
    name: string
    emoji?: string
    color?: string
    members: Record<string, string>
    automations?: number[]
    /** Id of the sync group the space is in; absent when it is in none. */
    sync_group?: number
}

export interface Manifest {
    /** RFC 3339: the "today" the data was seeded for. */
    now: string
    users: { key: string; name: string; email: string; password: string; role: string; verified: boolean }[]
    spaces: ManifestSpace[]
    invitations: { space_id: number; email?: string; role: string; token: string }[]
}
