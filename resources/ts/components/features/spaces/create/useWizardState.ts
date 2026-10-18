import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation } from '@tanstack/react-query'
import { toast } from 'sonner'
import { spacesApi } from '@/api/spaces'
import { getApiErrorMessage } from '@/lib/api-error'
import { useServerCurrencyCatalog, useSpaceContent, useSpacePresets, useSpaces } from '@/hooks/use-spaces'
import { useSpaceDraftStore } from '@/stores/space-draft'
import { Space } from '@/types/spaces'
import {
    buildContent,
    draftProblems,
    effectiveSourceId,
    effectiveStep,
    nextStep,
    toCopySource,
    toRequest,
    type DraftContent,
    type DraftStep,
} from './draft'
import { useLoadedDraft } from './useLoadedDraft'

/** The draft, what the steps read from the server, and the moves between steps. */
export function useWizardState({ open, onCreated }: { open: boolean; onCreated: (space: Space) => void }) {
    const { t } = useTranslation('settings')
    const draft = useSpaceDraftStore((state) => state.draft)
    const update = useSpaceDraftStore((state) => state.update)
    const clear = useSpaceDraftStore((state) => state.clear)
    const { data: spaces = [] } = useSpaces()
    const presets = useSpacePresets(open)
    const catalog = useServerCurrencyCatalog(open)
    const loaded = useLoadedDraft(open)

    const otherSpaces = spaces.length
    // Copying needs to read the other space; a source that is gone is ignored.
    const sourceId = effectiveSourceId(
        draft.sourceId,
        spaces.map((s) => s.id),
    )
    const source = useSpaceContent(open && draft.step === 'copy' ? sourceId : null)

    const step = effectiveStep(draft.step, otherSpaces)
    const problems = useMemo(() => draftProblems(draft), [draft])

    const create = useMutation({
        mutationFn: () => spacesApi.create(toRequest(draft)),
        onSuccess: (space) => {
            clear()
            onCreated(space)
        },
        onError: (error) => toast.error(getApiErrorMessage(error, t('spaces.create.failed'))),
    })

    const goTo = (to: DraftStep, patch: Partial<DraftContent> = {}) => update({ step: to, ...patch })
    const seed = () => {
        const target = nextStep(step, otherSpaces)
        const fromCopy = step === 'copy'
        const sourceContent = fromCopy && source.data ? toCopySource(source.data) : null
        // Leaving the name step for the entities does not discard edits made earlier.
        const seeded = draft.categories.length > 0 && !fromCopy
        goTo(target, seeded ? {} : buildContent(draft.copy, sourceContent, presets.data!))
    }

    return {
        draft,
        update,
        spaces,
        presets,
        catalog,
        loaded,
        sourceId,
        source,
        step,
        otherSpaces,
        problems,
        create,
        goTo,
        seed,
    }
}
