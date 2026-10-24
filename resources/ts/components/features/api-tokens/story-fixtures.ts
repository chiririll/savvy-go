import type { TokenCapabilities } from '@/types'

export const capabilities: TokenCapabilities = {
    levels: ['read', 'write', 'delete'],
    presets: ['viewer', 'editor'],
    kinds: {
        personal: {
            resources: ['transactions', 'accounts', 'categories'],
            presets: {
                viewer: ['transactions:read', 'accounts:read', 'categories:read'],
                editor: ['transactions:write', 'accounts:write', 'categories:write'],
            },
        },
        mcp: { resources: ['transactions'], presets: { viewer: ['transactions:read'], editor: ['transactions:write'] } },
    },
}
