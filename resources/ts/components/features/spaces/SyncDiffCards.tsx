import type { SyncDiff } from '@/types/spaces'
import { SyncCategoryCard, SyncDefaultCard, SyncTagCard } from './SyncReconcileCards'
import {
    categoryKey,
    defaultChoice,
    defaultOptions,
    isKept,
    needsSuccessor,
    successorNames,
    tagKey,
    type ReconcileChoices,
} from './sync-reconcile'
import type { Category } from '@/types'

type Choose = (patch: { [K in keyof ReconcileChoices]?: ReconcileChoices[K] }) => void

interface SyncDiffCardsProps {
    data: SyncDiff
    choices: ReconcileChoices
    onChoose: Choose
    /** Categories of the current space, offered as successors. */
    mine: Category[]
    spaceName: (id: number) => string
    names: (ids: number[]) => string
}

/** One card per difference between the spaces: categories, default categories and tags. */
export function SyncDiffCards({ data, choices, onChoose: choose, mine, spaceName, names }: SyncDiffCardsProps) {
    return (
        <div className="space-y-4">
            {data.categories.map((c) => {
                const key = categoryKey(c)
                return (
                    <SyncCategoryCard
                        key={key}
                        conflict={c}
                        kept={isKept(choices, key)}
                        look={choices.look[key]}
                        successor={choices.successor[key]}
                        needsSuccessor={needsSuccessor(c, choices)}
                        successors={successorNames(mine, data, c.type)}
                        spaceName={spaceName}
                        names={names}
                        onKeep={(kept) => choose({ keep: { ...choices.keep, [key]: kept } })}
                        onLook={(spaceId) => choose({ look: { ...choices.look, [key]: spaceId } })}
                        onSuccessor={(name) => choose({ successor: { ...choices.successor, [key]: name } })}
                    />
                )
            })}

            {data.defaults.map((d) => (
                <SyncDefaultCard
                    key={d.type}
                    conflict={d}
                    value={defaultChoice(choices, d.type, defaultOptions(d))}
                    spaceName={spaceName}
                    onChange={(name) => choose({ defaults: { ...choices.defaults, [d.type]: name } })}
                />
            ))}

            {data.tags.map((c) => {
                const key = tagKey(c)
                return (
                    <SyncTagCard
                        key={key}
                        conflict={c}
                        kept={isKept(choices, key)}
                        names={names}
                        onKeep={(kept) => choose({ keep: { ...choices.keep, [key]: kept } })}
                    />
                )
            })}
        </div>
    )
}
