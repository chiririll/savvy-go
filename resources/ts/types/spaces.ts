export type SpaceRole = 'admin' | 'editor' | 'viewer'

export interface Space {
    id: number
    uuid: string
    name: string
    role: SpaceRole
}
