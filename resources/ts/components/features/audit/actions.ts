import type { TFunction } from 'i18next'
import type { SpaceAction } from '@/types/spaces'

export const OPS = ['create', 'update', 'delete', 'confirm', 'skip', 'rollback', 'cleanup'] as const
type Op = (typeof OPS)[number]
export const ENTITIES = ['transaction', 'import', 'audit'] as const
type Entity = (typeof ENTITIES)[number]
export const SOURCES = ['ui', 'api', 'mcp', 'system'] as const
type Source = (typeof SOURCES)[number]

const known = <T extends string>(list: readonly T[], value: string): value is T => (list as readonly string[]).includes(value)

export const entityLabel = (t: TFunction, entity: Entity): string => ({
    transaction: t('settings:spaces.actions.entity.transaction'),
    import: t('settings:spaces.actions.entity.import'),
    audit: t('settings:spaces.actions.entity.audit'),
} satisfies Record<Entity, string>)[entity]

const opLabel = (t: TFunction, op: Op, entity: string): string => ({
    create: t('settings:spaces.actions.op.create', { entity }),
    update: t('settings:spaces.actions.op.update', { entity }),
    delete: t('settings:spaces.actions.op.delete', { entity }),
    confirm: t('settings:spaces.actions.op.confirm', { entity }),
    skip: t('settings:spaces.actions.op.skip', { entity }),
    rollback: t('settings:spaces.actions.op.rollback', { entity }),
    cleanup: t('settings:spaces.actions.op.cleanup', { entity }),
} satisfies Record<Op, string>)[op]

/** "Created transaction", "Cleared the log"; a value the app does not know yet is shown as the server sent it. */
export function actionTitle(t: TFunction, a: Pick<SpaceAction, 'op' | 'entityType'>): string {
    if (a.entityType === 'audit' && a.op === 'cleanup') return t('settings:spaces.actions.op.cleanupLog')
    if (!known(OPS, a.op)) return `${a.op} ${a.entityType}`
    return opLabel(t, a.op, known(ENTITIES, a.entityType) ? entityLabel(t, a.entityType) : a.entityType)
}

const sourceLabel = (t: TFunction, source: Source): string => ({
    ui: t('settings:spaces.actions.source.ui'),
    api: t('settings:spaces.actions.source.api'),
    mcp: t('settings:spaces.actions.source.mcp'),
    system: t('settings:spaces.actions.source.system'),
} satisfies Record<Source, string>)[source]

export function actionSourceLabel(t: TFunction, source: string): string {
    return known(SOURCES, source) ? sourceLabel(t, source) : source
}

/** One row of the list: a single entry, or the consecutive entries of one bulk call folded together. */
export type ActionGroup = { key: string; first: SpaceAction; items: SpaceAction[] }

/** Folds neighbours that share a batch, operation, entity type and author into one group; the order stays newest first. */
export function groupActions(actions: SpaceAction[]): ActionGroup[] {
    const groups: ActionGroup[] = []
    for (const a of actions) {
        const last = groups[groups.length - 1]
        const same =
            !!a.batchId &&
            last?.first.batchId === a.batchId &&
            last.first.op === a.op &&
            last.first.entityType === a.entityType &&
            last.first.userId === a.userId &&
            last.first.source === a.source
        if (same) last.items.push(a)
        else groups.push({ key: String(a.id), first: a, items: [a] })
    }
    return groups
}

/** What the log is narrowed to; an empty field is no filter. */
export interface ActionFilters {
    op?: string
    entityType?: string
    source?: string
}
