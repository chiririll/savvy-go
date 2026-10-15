// The pages of a space, as its first admin sees them straight after creating it.
import { pages } from '../../../../../resources/ts/app/pages.ts'
import { scenario } from '../../types.ts'
import { visit } from '../../visit.ts'
import { createSpace } from '../onboarding/create-space.ts'

const list = [
    ['dashboard', pages.dashboard],
    ['transactions', pages.transactions],
    ['accounts', pages.accounts],
    ['categories', pages.categories],
    ['currencies', pages.currencies],
    ['budgets', pages.budgets],
    ['tags', pages.tags],
    ['debts', pages.debts],
    ['recurring', pages.recurring],
    ['automation', pages.automation],
    ['reports', pages.reports],
    ['space-settings', pages.spaceSettings],
    ['import', pages.importSettings],
    ['backups', pages.spaceBackups],
] as const

export const spacePages = scenario(
    {
        name: 'pages/space',
        covers: list.map(([, p]) => p.path),
        shots: list.map(([name]) => name),
        leaves: 'the admin in the space "Family Budget", on its backups',
    },
    async ({ page, run, shot }) => {
        await run(createSpace)
        for (const [name, p] of list) {
            await visit(page, p.path)
            await shot(name)
        }
    },
)
