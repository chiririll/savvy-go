import axios from 'axios'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { uploadsApi } from '@/api/uploads'

import { multipartUpload } from './multipart-upload'

vi.mock('@/api/uploads', () => ({
    uploadsApi: {
        create: vi.fn(),
        signPart: vi.fn(),
        complete: vi.fn(),
        abort: vi.fn(),
    },
}))

vi.mock('axios', () => ({ default: { put: vi.fn() } }))

const put = vi.mocked(axios.put)
const api = vi.mocked(uploadsApi)

function fileOf(size: number): File {
    return new File([new Uint8Array(size)], 'data.csv', { type: 'text/csv' })
}

beforeEach(() => {
    vi.resetAllMocks()
    api.create.mockResolvedValue({
        uploadId: 'up1',
        key: 'imports/up1-data.csv',
        bucket: 'imports',
        partSize: 4,
        expiresAt: null,
    })
    api.signPart.mockImplementation(async (_id, _key, n) => ({
        url: `/api/uploads/up1/parts/${n}?signature=s`,
        method: 'PUT',
        headers: {},
        expires: 3600,
    }))
    api.complete.mockResolvedValue({ location: '/api/s3/imports/x', key: 'imports/up1-data.csv', uploadId: 'up1' })
    api.abort.mockResolvedValue(undefined)
    put.mockImplementation(async (url) => ({ headers: { etag: `"etag-${String(url).match(/parts\/(\d+)/)![1]}"` } }))
})

describe('multipartUpload', () => {
    it('uploads the file in server-sized parts and completes them in order', async () => {
        const result = await multipartUpload(fileOf(10), { bucket: 'imports' })

        expect(api.create).toHaveBeenCalledWith({
            bucket: 'imports',
            filename: 'data.csv',
            type: 'text/csv',
            size: 10,
        })
        expect(put).toHaveBeenCalledTimes(3)
        expect(api.complete).toHaveBeenCalledWith('up1', 'imports/up1-data.csv', [
            { PartNumber: 1, ETag: 'etag-1' },
            { PartNumber: 2, ETag: 'etag-2' },
            { PartNumber: 3, ETag: 'etag-3' },
        ])
        expect(result).toEqual({ uploadId: 'up1', key: 'imports/up1-data.csv', location: '/api/s3/imports/x' })
    })

    it('sends an empty file as a single empty part', async () => {
        await multipartUpload(fileOf(0), { bucket: 'imports' })

        expect(put).toHaveBeenCalledTimes(1)
        expect(api.complete).toHaveBeenCalledWith('up1', 'imports/up1-data.csv', [{ PartNumber: 1, ETag: 'etag-1' }])
    })

    it('never runs more parts at once than the concurrency allows', async () => {
        let running = 0
        let peak = 0
        put.mockImplementation(async (url) => {
            running++
            peak = Math.max(peak, running)
            await new Promise((resolve) => setTimeout(resolve, 5))
            running--
            return { headers: { etag: `"e${String(url).length}"` } }
        })

        await multipartUpload(fileOf(40), { bucket: 'imports', concurrency: 2 })

        expect(peak).toBe(2)
    })

    it('reports progress that ends at 100 percent', async () => {
        const onProgress = vi.fn()
        put.mockImplementation(async (url, data, config) => {
            const size = (data as Blob).size
            config?.onUploadProgress?.({ loaded: size, total: size, bytes: size } as never)
            return { headers: { etag: `"e${String(url).length}"` } }
        })

        await multipartUpload(fileOf(10), { bucket: 'imports', onProgress })

        expect(onProgress).toHaveBeenLastCalledWith({ loaded: 10, total: 10, percentage: 100 })
    })

    it('aborts the server upload and rejects when a part fails', async () => {
        put.mockRejectedValueOnce(new Error('network down'))

        await expect(multipartUpload(fileOf(10), { bucket: 'imports', concurrency: 1 })).rejects.toThrow(
            'network down'
        )

        expect(api.abort).toHaveBeenCalledWith('up1', 'imports/up1-data.csv')
        expect(api.complete).not.toHaveBeenCalled()
    })

    it('aborts the server upload when the signal fires', async () => {
        const controller = new AbortController()
        put.mockImplementation(async (_url, _data, config) => {
            controller.abort()
            const signal = config?.signal as AbortSignal
            signal.throwIfAborted()
            return { headers: { etag: '"x"' } }
        })

        await expect(
            multipartUpload(fileOf(10), { bucket: 'imports', concurrency: 1, signal: controller.signal })
        ).rejects.toThrow()

        expect(api.abort).toHaveBeenCalledWith('up1', 'imports/up1-data.csv')
    })

    it('does not start when the signal is already aborted', async () => {
        const controller = new AbortController()
        controller.abort()

        await expect(multipartUpload(fileOf(10), { bucket: 'imports', signal: controller.signal })).rejects.toThrow()

        expect(api.create).not.toHaveBeenCalled()
    })

    it('fails when a part comes back without an ETag', async () => {
        put.mockResolvedValue({ headers: {} })

        await expect(multipartUpload(fileOf(3), { bucket: 'imports' })).rejects.toThrow('ETag')
        expect(api.abort).toHaveBeenCalled()
    })
})
