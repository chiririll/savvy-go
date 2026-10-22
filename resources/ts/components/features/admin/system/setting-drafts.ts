export interface NumberDraft {
    /** What would be saved; null for an empty field. */
    parsed: number | null
    valid: boolean
    changed: boolean
}

/** A number field is valid when it is a non-negative number, or empty where null is allowed. */
export function numberDraft(draft: string, saved: number | null, nullable = false): NumberDraft {
    const parsed = draft.trim() === '' ? null : Number(draft)
    const valid = (parsed === null && nullable) || (parsed !== null && Number.isFinite(parsed) && parsed >= 0)
    return { parsed, valid, changed: parsed !== saved }
}

export const numberToDraft = (value: number | null) => (value === null ? '' : String(value))
