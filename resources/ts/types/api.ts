export interface BaseEntity {
    id: number
    createdAt?: string
    updatedAt?: string
}

export interface PaginatedResponse<T> {
    data: T[]
    total: number
    page: number
    limit: number
}

export interface ApiError {
    message: string
    code: string
    /** The server's machine-readable error code, when it sent one (`confirmation_invalid`). */
    reason?: string
    details?: Record<string, string[]>
}

export interface CrudConfig {
    endpoint: string
    queryKey: string[]
    redirectTo?: string
}

export type MutationData<T> = Omit<T, 'id' | 'createdAt' | 'updatedAt'>
