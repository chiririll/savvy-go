// The settings of the signed-in user: security, passkeys and API tokens.
import { pages } from '../../../../../resources/ts/app/pages.ts'
import { scenario } from '../../types.ts'
import { visit } from '../../visit.ts'
import { createSpace } from '../onboarding/create-space.ts'

const list = [
    ['security', pages.securitySettings],
    ['passkeys', pages.passkeySettings],
    ['api', pages.apiSettings],
] as const

export const userPages = scenario(
    {
        name: 'pages/user',
        covers: list.map(([, p]) => p.path),
        shots: list.map(([name]) => name),
        leaves: 'the admin in the space "Family Budget", on the API settings',
    },
    async ({ page, run, shot }) => {
        await run(createSpace)
        for (const [name, p] of list) {
            await visit(page, p.path)
            await shot(name)
        }
    },
)
