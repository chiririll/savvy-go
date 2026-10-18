import { type ReactNode } from 'react'
import { Loader2 } from 'lucide-react'
import { Space } from '@/types/spaces'
import { CreateSubmitButton } from './CreateSubmitButton'
import { ConfirmStep } from './ConfirmStep'
import { BackButton, CancelButton, NextButton } from './WizardButtons'
import { CopyStep } from './CopyStep'
import { EntitiesStep } from './EntitiesStep'
import { NameStep } from './NameStep'
import { previousStep, toRequest } from './draft'
import { useWizardState } from './useWizardState'

interface Options {
    open: boolean
    onClose: () => void
    /** A user without any space cannot close the modal. */
    dismissible: boolean
    onCreated: (space: Space) => void
}

/** The body and footer of the current step of the create-space wizard, up to sending the request. */
export function useWizardStep({ open, onClose, dismissible, onCreated }: Options): {
    body: ReactNode
    footer: ReactNode
} {
    const {
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
    } = useWizardState({ open, onCreated })

    const back = <BackButton visible={step !== 'name'} onClick={() => goTo(previousStep(step, otherSpaces))} />
    const cancel = <CancelButton visible={dismissible} onClick={onClose} />
    const next = (disabled: boolean, onClick: () => void) => <NextButton disabled={disabled} onClick={onClick} />

    // Only the stored draft is needed to show a step. The presets are needed to leave the
    // name/copy step, and the currency catalog (which may wait on the rate source) only by the currency selects.
    if (!loaded) {
        return { body: <Loader2 className="mx-auto size-6 animate-spin text-muted-foreground" />, footer: cancel }
    }

    switch (step) {
        case 'name':
            return {
                body: (
                    <NameStep
                        name={draft.name}
                        onNameChange={(name) => update({ name })}
                        emoji={draft.emoji}
                        color={draft.color}
                        onLookChange={update}
                        onSubmit={seed}
                    />
                ),
                footer: (
                    <>
                        {cancel}
                        {next(!draft.name.trim() || !presets.data, seed)}
                    </>
                ),
            }
        case 'copy':
            return {
                body: (
                    <CopyStep
                        spaces={spaces}
                        sourceId={sourceId}
                        copy={draft.copy}
                        onSourceChange={(id) => update({ sourceId: id })}
                        onCopyChange={(copy) => update({ copy })}
                    />
                ),
                footer: (
                    <>
                        {back}
                        {next(!presets.data || (sourceId !== null && source.isPending), seed)}
                    </>
                ),
            }
        case 'entities':
            return {
                body: (
                    <EntitiesStep
                        content={draft}
                        catalog={catalog.data}
                        catalogFailed={catalog.isError}
                        onRetryCatalog={() => void catalog.refetch()}
                        onChange={(patch) => update(patch)}
                    />
                ),
                footer: (
                    <>
                        {back}
                        {next(
                            !draft.baseCurrency || problems.includes('accounts') || problems.includes('balances'),
                            () => goTo('confirm'),
                        )}
                    </>
                ),
            }
        default:
            return {
                body: <ConfirmStep request={toRequest(draft)} />,
                footer: (
                    <>
                        {back}
                        <CreateSubmitButton
                            disabled={problems.length > 0}
                            pending={create.isPending}
                            onClick={() => create.mutate()}
                        />
                    </>
                ),
            }
    }
}
