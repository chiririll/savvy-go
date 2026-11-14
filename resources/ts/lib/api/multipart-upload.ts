import axios from 'axios'
import { uploadsApi } from '@/api/uploads'
import type { CompletedPart, MultipartUploadResult, UploadProgress } from '@/types/upload'

interface MultipartUploadOptions {
    bucket: string
    concurrency?: number
    signal?: AbortSignal
    onProgress?: (progress: UploadProgress) => void
}

const DEFAULT_PART_SIZE = 8 * 1024 * 1024

function stripQuotes(etag: string): string {
    return etag.replace(/^"|"$/g, '')
}

/**
 * Uploads a file through the backend's multipart API: create, one signed PUT
 * per part, complete. The server owns part sizing, so the file is cut by the
 * `partSize` it returns. A failure or abort cancels the server-side upload.
 */
export async function multipartUpload(
    file: File,
    options: MultipartUploadOptions
): Promise<MultipartUploadResult> {
    const { bucket, concurrency = 4, signal, onProgress } = options

    signal?.throwIfAborted()

    const created = await uploadsApi.create({
        bucket,
        filename: file.name,
        type: file.type,
        size: file.size,
    })

    const partSize = created.partSize > 0 ? created.partSize : DEFAULT_PART_SIZE
    const partCount = Math.max(1, Math.ceil(file.size / partSize))
    const loadedByPart = new Array<number>(partCount).fill(0)
    const completed: CompletedPart[] = new Array(partCount)

    const report = () => {
        const loaded = loadedByPart.reduce((sum, n) => sum + n, 0)
        onProgress?.({
            loaded,
            total: file.size,
            percentage: file.size ? Math.min(100, Math.round((loaded / file.size) * 100)) : 100,
        })
    }

    // Aborts the in-flight parts as soon as one fails, so the others stop too.
    const failure = new AbortController()
    const onAbort = () => failure.abort(signal?.reason)
    signal?.addEventListener('abort', onAbort, { once: true })

    const uploadPart = async (index: number) => {
        const partNumber = index + 1
        const blob = file.slice(index * partSize, (index + 1) * partSize)
        const signed = await uploadsApi.signPart(created.uploadId, created.key, partNumber)

        const response = await axios.put(signed.url, blob, {
            headers: signed.headers,
            signal: failure.signal,
            onUploadProgress: (event) => {
                loadedByPart[index] = Math.min(event.loaded, blob.size)
                report()
            },
        })

        const etag = response.headers?.etag
        if (typeof etag !== 'string' || etag === '') {
            throw new Error('Upload part came back without an ETag')
        }

        loadedByPart[index] = blob.size
        report()
        completed[index] = { PartNumber: partNumber, ETag: stripQuotes(etag) }
    }

    let next = 0
    const worker = async () => {
        while (next < partCount) {
            failure.signal.throwIfAborted()
            await uploadPart(next++)
        }
    }

    try {
        const workers = Array.from({ length: Math.min(concurrency, partCount) }, () =>
            worker().catch((error: unknown) => {
                failure.abort(error)
                throw error
            })
        )
        await Promise.all(workers)

        const result = await uploadsApi.complete(created.uploadId, created.key, completed)

        return { uploadId: created.uploadId, key: created.key, location: result.location }
    } catch (error) {
        await uploadsApi.abort(created.uploadId, created.key).catch(() => undefined)
        throw error
    } finally {
        signal?.removeEventListener('abort', onAbort)
    }
}
