import { apiClient } from './client'

export const onboardingApi = {
    /**
     * Restores a server backup (or a Laravel database, with its app key) on a server that has no users yet.
     * The bootstrap password authorizes it; nobody is signed in afterwards.
     */
    restore: async (bootstrapPassword: string, file: File, appKey?: string): Promise<void> => {
        const formData = new FormData()
        formData.append('bootstrap_password', bootstrapPassword)
        formData.append('file', file)
        if (appKey) formData.append('app_key', appKey)
        await apiClient.post('/onboarding/restore', formData, {
            headers: { 'Content-Type': 'multipart/form-data' },
            timeout: 0,
        })
    },
}
