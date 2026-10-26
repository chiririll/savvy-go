/**
 * The items with those ids, in that order, put back into the places the same
 * ids held in the list. The rest of the list stays where it is.
 */
export function placeInSlots<T extends { id: number }>(items: T[], orderedIds: number[]): T[] {
    const byId = new Map(items.map((item) => [item.id, item]))
    const moved = orderedIds.map((id) => byId.get(id)).filter((item): item is T => item !== undefined)
    const movedIds = new Set(moved.map((item) => item.id))
    let next = 0

    return items.map((item) => (movedIds.has(item.id) ? moved[next++] : item))
}

/** The server takes a day's order oldest first; a list sorted newest first shows it reversed. */
export function dayOrderForApi(shownIds: number[], direction: 'asc' | 'desc'): number[] {
    return direction === 'desc' ? [...shownIds].reverse() : shownIds
}
