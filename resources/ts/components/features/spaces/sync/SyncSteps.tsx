import { useTranslation } from 'react-i18next'
import type { Category } from '@/types'
import type { SyncDiff } from '@/types/spaces'
import { SyncCategoryCard, SyncDefaultCard, SyncSuccessorCard, SyncTagCard } from './SyncReconcileCards'
import {
    categoryKey,
    defaultChoice,
    defaultOptions,
    isKept,
    needsSuccessor,
    successorNames,
    summarize,
    tagKey,
    type ReconcileChoices,
    type SyncStep,
} from './sync-reconcile'

type Choose = (patch: { [K in keyof ReconcileChoices]?: ReconcileChoices[K] }) => void

interface Props {
    step: SyncStep
    data: SyncDiff
    choices: ReconcileChoices
    onChoose: Choose
    /** Categories of the current space, offered as successors. */
    mine: Category[]
    spaceName: (id: number) => string
    names: (ids: number[]) => string
}

/** The body of one step of the reconcile dialog: the cards for what that step decides. */
export function SyncStepBody({ step, data, choices, onChoose: choose, mine, spaceName, names }: Props) {
    const { t } = useTranslation('settings')

    switch (step) {
        case 'categories':
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
                                spaceName={spaceName}
                                names={names}
                                onKeep={(kept) => choose({ keep: { ...choices.keep, [key]: kept } })}
                                onLook={(spaceId) => choose({ look: { ...choices.look, [key]: spaceId } })}
                            />
                        )
                    })}
                </div>
            )
        case 'merge':
            return (
                <div className="space-y-4">
                    {data.categories
                        .filter((c) => needsSuccessor(c, choices))
                        .map((c) => {
                            const key = categoryKey(c)
                            return (
                                <SyncSuccessorCard
                                    key={key}
                                    conflict={c}
                                    value={choices.successor[key]}
                                    options={successorNames(mine, data, c.type)}
                                    onChange={(name) => choose({ successor: { ...choices.successor, [key]: name } })}
                                />
                            )
                        })}
                </div>
            )
        case 'tags':
            return (
                <div className="space-y-4">
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
        default: {
            const sum = summarize(data, choices)
            return (
                <div className="space-y-4">
                    {data.defaults.map((d) => (
                        <SyncDefaultCard
                            key={d.type}
                            conflict={d}
                            value={defaultChoice(choices, d.type, defaultOptions(d))}
                            spaceName={spaceName}
                            onChange={(name) => choose({ defaults: { ...choices.defaults, [d.type]: name } })}
                        />
                    ))}
                    <ul className="space-y-1 rounded-lg border bg-muted/30 p-3 text-sm">
                        {data.categories.length > 0 && (
                            <li>
                                {t('spaces.sync.dialog.summaryCategories', {
                                    kept: sum.categoriesKept,
                                    dropped: sum.categoriesDropped,
                                })}
                            </li>
                        )}
                        {data.tags.length > 0 && (
                            <li>
                                {t('spaces.sync.dialog.summaryTags', { kept: sum.tagsKept, dropped: sum.tagsDropped })}
                            </li>
                        )}
                        {data.categories.length === 0 && data.tags.length === 0 && (
                            <li>{t('spaces.sync.dialog.summaryDefaults')}</li>
                        )}
                    </ul>
                </div>
            )
        }
    }
}
