// What the demo seed writes to SEED_MANIFEST; see internal/seed/manifest.go.
export interface ManifestSpace {
    id: number
    name: string
    members: Record<string, string>
    automations?: number[]
}

export interface Manifest {
    /** RFC 3339: the "today" the data was seeded for. */
    now: string
    users: { key: string; name: string; email: string; password: string; role: string }[]
    spaces: ManifestSpace[]
    invitations: { space_id: number; email?: string; role: string; token: string }[]
}
