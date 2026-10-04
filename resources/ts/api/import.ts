import { apiClient } from './client'
import type {
    ColumnMapping,
    ImportOptions,
    ImportPreviewResult,
    ImportState,
} from '@/types/import'

const ENDPOINT = '/transactions/import'

// Requests stay snake_case like the rest of the API input.
const optionsToSnakeCase = (options: ImportOptions) => ({
    date_format: options.dateFormat,
    amount_format: options.amountFormat,
    default_account_id: options.defaultAccountId,
    default_type: options.defaultType,
    skip_first_row: options.skipFirstRow,
    create_missing_currencies: options.createMissingCurrencies,
    create_missing_tags: options.createMissingTags,
    create_missing_categories: options.createMissingCategories,
    category_map: options.categoryMap ?? {},
})

const delay = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

export const importApi = {
    parse: async (uploadId: string): Promise<ImportState> => {
        const response = await apiClient.post(`${ENDPOINT}/parse`, { upload_id: uploadId })

        return response.data.data as ImportState
    },

    status: async (importId: string): Promise<ImportState> => {
        const response = await apiClient.get(`${ENDPOINT}/${importId}`)

        return response.data.data as ImportState
    },

    preview: async (
        importId: string,
        mapping: ColumnMapping,
        options: ImportOptions
    ): Promise<ImportPreviewResult> => {
        const response = await apiClient.post(`${ENDPOINT}/preview`, {
            import_id: importId,
            mapping,
            options: optionsToSnakeCase(options),
        })

        return response.data.data as ImportPreviewResult
    },

    execute: async (
        importId: string,
        mapping: ColumnMapping,
        options: ImportOptions
    ): Promise<ImportState> => {
        const response = await apiClient.post(`${ENDPOINT}/execute`, {
            import_id: importId,
            mapping,
            options: optionsToSnakeCase(options),
        })

        return response.data.data as ImportState
    },

    poll: async (
        importId: string,
        until: (state: ImportState) => boolean,
        onTick?: (state: ImportState) => void,
        intervalMs = 1000
    ): Promise<ImportState> => {
        while (true) {
            const state = await importApi.status(importId)
            onTick?.(state)

            if (state.status === 'failed') {
                throw new Error(state.message || 'Import failed')
            }

            if (until(state)) {
                return state
            }

            await delay(intervalMs)
        }
    },
}
