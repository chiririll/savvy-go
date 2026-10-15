import { createDraftStore } from '@/lib/draft-store'
import { DRAFT_STEPS, emptyDraft } from '@/components/features/spaces/create/draft'

export const spaceDraft = createDraftStore({
    key: 'savvy-space-draft',
    empty: emptyDraft,
    accept: (draft) => DRAFT_STEPS.includes(draft.step),
})

export const useSpaceDraftStore = spaceDraft.useStore

/** Loads the draft of the signed-in user; call when the modal opens. */
export const loadSpaceDraft = spaceDraft.load
