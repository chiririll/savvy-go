import { useMemo } from 'react'
import { AbilityProvider as CaslAbilityProvider, Can as CaslCan, useAbility as useCaslAbility, type CanProps } from '@casl/react'
import { defineAbilityFor, AppAbility } from '@/lib/ability'
import { useUser } from '@/stores/auth'

export const Can: (props: CanProps<AppAbility>) => React.ReactNode = CaslCan

export function AbilityProvider({ children }: { children: React.ReactNode }) {
    const user = useUser()
    const ability = useMemo(() => defineAbilityFor(user?.role ?? null), [user?.role])

    return <CaslAbilityProvider value={ability}>{children}</CaslAbilityProvider>
}

export const useAbility = () => useCaslAbility<AppAbility>()
