import type { SyncCategoryConflict, SyncDecision, SyncDiff, SyncTagConflict } from '@/types/spaces'

/** What the user picked per difference; anything not picked takes the harmless side. */
export interface ReconcileChoices {
    keep: Record<string, boolean>
    look: Record<string, number>
    successor: Record<string, string>
    defaults: Record<string, string>
}

export const emptyChoices = (): ReconcileChoices => ({ keep: {}, look: {}, successor: {}, defaults: {} })

export type SyncDefaultConflict = SyncDiff['defaults'][number]

export const categoryKey = (c: Pick<SyncCategoryConflict, 'type' | 'name'>) => `c:${c.type}:${c.name.toLowerCase()}`
export const tagKey = (c: Pick<SyncTagConflict, 'name'>) => `t:${c.name.toLowerCase()}`

/** Whether the spaces give the category different looks, so one has to be picked. */
export const differs = (c: SyncCategoryConflict) =>
    c.present.some(
        (p) => (p.icon ?? '') !== (c.present[0].icon ?? '') || (p.color ?? '') !== (c.present[0].color ?? ''),
    )

export const isKept = (choices: ReconcileChoices, key: string) => choices.keep[key] ?? true

/** A dropped category that holds transactions needs another one to take them. */
export const needsSuccessor = (c: SyncCategoryConflict, choices: ReconcileChoices) =>
    !isKept(choices, categoryKey(c)) && c.present.some((p) => p.transactions > 0)

export const defaultOptions = (d: SyncDefaultConflict) => [...new Set(d.defaults.map((x) => x.name).filter(Boolean))]

export const defaultChoice = (choices: ReconcileChoices, type: string, options: string[]) =>
    choices.defaults[type] ?? options[0]

/** Categories every space has already: where the transactions of a dropped one can go. */
export function successorNames(
    mine: Pick<SyncCategoryConflict, 'type' | 'name'>[],
    data: SyncDiff | undefined,
    type: string,
) {
    const conflicting = new Set((data?.categories ?? []).map(categoryKey))
    return mine.filter((c) => c.type === type && !conflicting.has(categoryKey(c))).map((c) => c.name)
}

export function buildDecisions(data: SyncDiff | undefined, choices: ReconcileChoices): SyncDecision[] {
    return [
        ...(data?.categories ?? []).map((c): SyncDecision => {
            const key = categoryKey(c)
            const kept = isKept(choices, key)
            return {
                kind: 'categories',
                type: c.type,
                name: c.name,
                keep: kept,
                from_space_id: kept && differs(c) ? (choices.look[key] ?? c.present[0].spaceId) : undefined,
                successor: !kept ? choices.successor[key] : undefined,
            }
        }),
        ...(data?.tags ?? []).map((c): SyncDecision => ({
            kind: 'tags',
            name: c.name,
            keep: isKept(choices, tagKey(c)),
        })),
    ]
}

export const buildDefaultDecisions = (data: SyncDiff | undefined, choices: ReconcileChoices) =>
    (data?.defaults ?? []).map((d) => ({ type: d.type, name: defaultChoice(choices, d.type, defaultOptions(d)) }))

export const isComplete = (data: SyncDiff | undefined, choices: ReconcileChoices) =>
    !!data && data.categories.every((c) => !needsSuccessor(c, choices) || !!choices.successor[categoryKey(c)])

export const isIdentical = (data: SyncDiff | undefined) =>
    !!data && !data.categories.length && !data.tags.length && !data.defaults.length

export type SyncStep = 'categories' | 'merge' | 'tags' | 'summary'

/**
 * The steps of the reconcile dialog, in order. A step with nothing to decide is left out; the merge
 * step (where the transactions of a deleted category go) exists only while a choice needs it.
 */
export function syncSteps(data: SyncDiff | undefined, choices: ReconcileChoices): SyncStep[] {
    const steps: SyncStep[] = []
    if (data?.categories.length) steps.push('categories')
    if (data?.categories.some((c) => needsSuccessor(c, choices))) steps.push('merge')
    if (data?.tags.length) steps.push('tags')
    return [...steps, 'summary']
}

/** What applying the choices does, for the last step. */
export function summarize(data: SyncDiff | undefined, choices: ReconcileChoices) {
    const categories = data?.categories ?? []
    const tags = data?.tags ?? []
    const kept = categories.filter((c) => isKept(choices, categoryKey(c))).length
    const keptTags = tags.filter((c) => isKept(choices, tagKey(c))).length
    return {
        categoriesKept: kept,
        categoriesDropped: categories.length - kept,
        tagsKept: keptTags,
        tagsDropped: tags.length - keptTags,
    }
}
