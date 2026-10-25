import { createDraftStore } from '@/lib/draft-store'
import { emptyDraft, migrateStep } from '@/components/features/spaces/create/draft'

export const spaceDraft = createDraftStore({
    key: 'savvy-space-draft',
    empty: emptyDraft,
    // A step of an older version (`entities`) is still accepted; loadSpaceDraft moves it on.
    accept: (draft) => migrateStep(draft.step) !== null,
})

export const useSpaceDraftStore = spaceDraft.useStore

/** Loads the draft of the signed-in user; call when the modal opens. */
export async function loadSpaceDraft(): Promise<void> {
    await spaceDraft.load()
    const { draft, update } = spaceDraft.useStore.getState()
    const step = migrateStep(draft.step)
    if (step !== null && step !== draft.step) update({ step })
}
