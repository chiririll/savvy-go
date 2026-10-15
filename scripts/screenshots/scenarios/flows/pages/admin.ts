// The administration of the server. The edit page of a provider has its own scenario (a provider
// has to exist first).
import { pages } from '../../../../../resources/ts/app/pages.ts'
import { scenario } from '../../types.ts'
import { visit } from '../../visit.ts'
import { createSpace } from '../onboarding/create-space.ts'

const list = [
    ['system', pages.adminSystem],
    ['monitoring', pages.adminMonitoring],
    ['sso', pages.adminSso],
    ['security', pages.adminSecurity],
    ['provider-create', pages.providerCreate],
    ['backups', pages.adminBackups],
    ['users', pages.adminUsers],
    ['spaces', pages.adminSpaces],
] as const

export const adminPages = scenario(
    {
        name: 'pages/admin',
        covers: list.map(([, p]) => p.path),
        shots: list.map(([name]) => name),
        leaves: 'the admin in the space "Family Budget", on the list of spaces',
    },
    async ({ page, run, shot }) => {
        await run(createSpace)
        for (const [name, p] of list) {
            await visit(page, p.path)
            await shot(name)
        }
    },
)
