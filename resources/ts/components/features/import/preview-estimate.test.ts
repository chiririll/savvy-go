import { describe, expect, it } from 'vitest'

import type { ImportPreviewResult } from '@/types/import'
import { previewEstimate } from './preview-estimate'

const summary = (patch: Partial<ImportPreviewResult['summary']>) =>
    ({ willCreate: 40, sampled: 100, totalRows: 100, ...patch }) as ImportPreviewResult['summary']

describe('previewEstimate', () => {
    it('counts what will be created when the whole file was previewed', () => {
        expect(previewEstimate(summary({}))).toEqual({ isSampled: false, createEstimate: 40 })
    })

    it('estimates by the whole file when only a sample was previewed', () => {
        expect(previewEstimate(summary({ sampled: 100, totalRows: 5000 }))).toEqual({
            isSampled: true,
            createEstimate: 5000,
        })
    })

    it('treats an unknown row count as not sampled', () => {
        expect(previewEstimate(summary({ totalRows: null }))).toEqual({ isSampled: false, createEstimate: 40 })
    })
})
