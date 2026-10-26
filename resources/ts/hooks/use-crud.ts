import { useQuery, useMutation, useQueryClient, type QueryKey } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'

interface UseResourceMutationOptions<TData, TVariables> {
    mutationFn: (variables: TVariables) => Promise<TData>
    invalidateKeys?: QueryKey[]
    resetKeys?: QueryKey[]
    successMessage?: string | ((data: TData, variables: TVariables) => string)
    redirectTo?: string
    invalidateAll?: boolean
    /** For state outside react-query, e.g. the session user in the auth store. */
    onSuccess?: (data: TData, variables: TVariables) => void
}

export function useResourceMutation<TData, TVariables = void>({
    mutationFn,
    invalidateKeys = [],
    resetKeys = [],
    successMessage,
    redirectTo,
    invalidateAll = false,
    onSuccess,
}: UseResourceMutationOptions<TData, TVariables>) {
    const queryClient = useQueryClient()
    const navigate = useNavigate()

    return useMutation({
        mutationFn,
        onSuccess: (data, variables) => {
            if (invalidateAll) {
                queryClient.invalidateQueries()
            } else {
                resetKeys.forEach((queryKey) => {
                    queryClient.resetQueries({ queryKey })
                })
                invalidateKeys.forEach((queryKey) => {
                    queryClient.invalidateQueries({ queryKey })
                })
            }

            onSuccess?.(data, variables)

            if (successMessage) {
                toast.success(
                    typeof successMessage === 'function'
                        ? successMessage(data, variables)
                        : successMessage
                )
            }

            if (redirectTo) {
                navigate(redirectTo)
            }
        },
    })
}

export function useResourceItem<T>(
    queryKey: QueryKey,
    queryFn: () => Promise<T>,
    id: string | number,
) {
    return useQuery({
        queryKey: [...queryKey, id],
        queryFn,
        enabled: !!id,
    })
}
