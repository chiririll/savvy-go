import { createContext, useContext } from 'react'
import { useCurrentSpace } from '@/hooks/use-spaces'
import { canWriteSpace } from '@/lib/ability'

const ReadOnlyContext = createContext(false)

export function ReadOnlyProvider({ children }: { children: React.ReactNode }) {
    // Writing is decided by the role in the space, not the server role.
    const space = useCurrentSpace()
    const isReadOnly = space !== null && !canWriteSpace(space.role)

    return (
        <ReadOnlyContext.Provider value={isReadOnly}>
            {children}
        </ReadOnlyContext.Provider>
    )
}

export const useReadOnly = () => useContext(ReadOnlyContext)
