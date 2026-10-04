import { createContext, useContext } from 'react'
import { useCurrentSpace } from '@/hooks/use-spaces'

const ReadOnlyContext = createContext(false)

export function ReadOnlyProvider({ children }: { children: React.ReactNode }) {
    // Writing is decided by the role in the space, not the server role.
    const space = useCurrentSpace()
    const isReadOnly = space?.role === 'viewer'

    return (
        <ReadOnlyContext.Provider value={isReadOnly}>
            {children}
        </ReadOnlyContext.Provider>
    )
}

export const useReadOnly = () => useContext(ReadOnlyContext)
