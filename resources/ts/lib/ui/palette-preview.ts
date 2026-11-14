/** Results the palette lists before it offers "show all". */
export const PREVIEW_SIZE = 8
/** The "show all" row appears only when more than this many are left; fewer are listed instead. */
export const MAX_HIDDEN = 5

/** How many results the palette asks for; the server says whether more exist, without counting them. */
export const PREVIEW_FETCH = PREVIEW_SIZE + MAX_HIDDEN

/**
 * How many of the fetched results to list, and the "more than N" the "show all" row states (0: no
 * row). A short tail is listed whole, so the row appears only when more than `PREVIEW_FETCH` match.
 */
export function previewSplit(hasMore: boolean, fetched: number): { shown: number; moreThan: number } {
    if (!hasMore) {
        return { shown: fetched, moreThan: 0 }
    }
    return { shown: Math.min(PREVIEW_SIZE, fetched), moreThan: fetched }
}
