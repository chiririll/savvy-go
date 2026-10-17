import { readdirSync, readFileSync } from 'node:fs'
import path from 'node:path'
import { describe, expect, it } from 'vitest'

import { ACCOUNT_TYPES } from '@/types/accounts'
import { API_PERMISSION_LEVELS } from '@/types/api-tokens'
import { ACTION_TYPES, CONDITION_OPERATORS, TRIGGER_TYPES } from '@/types/automation'
import { BACKUP_STATUSES } from '@/types/backup'
import { BUDGET_PERIODS } from '@/types/budgets'
import { CHECK_IDS, CHECK_SEVERITIES } from '@/types/checks'
import { CATEGORY_TYPES } from '@/types/categories'
import { DEBT_TYPE_VALUES } from '@/types/debts'
import { MAIL_PROTOCOLS } from '@/types/mail'
import { RECURRING_FREQUENCIES } from '@/types/recurring'
import { SPACE_ROLES, TRANSFER_REVIEWS } from '@/types/spaces'
import { ALL_TRANSACTION_TYPES, TRANSACTION_STATUSES } from '@/types/transactions'
import { USER_ROLES } from '@/types/users'
import { canWriteSpace } from '@/lib/ability'
import { SSO_ERROR_CODES } from '@/lib/labels'
import { SPACE_ROOTS } from '@/lib/space-path'
import enSettings from '@/locales/en/settings.json'

/*
 * The frontend repeats a number of values the Go backend defines: enum columns,
 * error codes, API routes. Nothing ties the two together at build time, so
 * these tests read the Go sources and migrations and fail when they drift.
 */

const REPO = path.resolve(__dirname, '../../..')
// Line endings normalised: a Windows checkout may have CRLF.
const read = (file: string) => readFileSync(path.join(REPO, file), 'utf8').replace(/\r\n/g, '\n')
const sorted = (values: Iterable<string>) => [...new Set(values)].sort()

function goFiles(dir: string): string[] {
    return readdirSync(path.join(REPO, dir), { withFileTypes: true }).flatMap((entry) => {
        const rel = `${dir}/${entry.name}`
        if (entry.isDirectory()) return goFiles(rel)
        return entry.name.endsWith('.go') && !entry.name.endsWith('_test.go') ? [rel] : []
    })
}

/** The text of a Go function, from its signature to the closing brace in column 0. */
function goFunc(file: string, signature: string): string {
    const source = read(file)
    const start = source.indexOf(signature)
    if (start === -1) throw new Error(`${signature} not found in ${file}`)
    const end = source.indexOf('\n}\n', start)
    if (end === -1) throw new Error(`end of ${signature} not found in ${file}`)
    return source.slice(start, end)
}

const quoted = (text: string) => [...text.matchAll(/"([a-z_]+)"/g)].map((m) => m[1])

/** The values of a `CHECK (column IN ('a', 'b'))` constraint, per `table.column`. */
function sqlEnums(file: string): Map<string, string[]> {
    const enums = new Map<string, string[]>()
    for (const table of read(file).split(/CREATE TABLE/i).slice(1)) {
        const name = /^\s*(?:IF NOT EXISTS\s+)?(\w+)/.exec(table)![1]
        for (const m of table.matchAll(/(\w+)\s+TEXT[^\n]*?CHECK\s*\(\s*\1\s+IN\s*\(([^)]*)\)\s*\)/g)) {
            enums.set(`${name}.${m[1]}`, sorted(m[2].match(/'([^']*)'/g)!.map((v) => v.slice(1, -1))))
        }
    }
    return enums
}

const serverSchema = sqlEnums('internal/migrate/sql/server/0001_initial.sql')
const spaceSchema = sqlEnums('internal/migrate/sql/space/0001_initial.sql')

describe('enum columns in the database schema', () => {
    const cases: Array<[string, Map<string, string[]>, string, readonly string[]]> = [
        ['users.role', serverSchema, 'users.role', USER_ROLES],
        ['space_members.role', serverSchema, 'space_members.role', SPACE_ROLES],
        ['space_invitations.role', serverSchema, 'space_invitations.role', SPACE_ROLES],
        ['accounts.type', spaceSchema, 'accounts.type', ACCOUNT_TYPES],
        ['accounts.debt_type', spaceSchema, 'accounts.debt_type', DEBT_TYPE_VALUES],
        ['categories.type', spaceSchema, 'categories.type', CATEGORY_TYPES],
        ['recurring_transactions.frequency', spaceSchema, 'recurring_transactions.frequency', RECURRING_FREQUENCIES],
        ['space_transfers.review', spaceSchema, 'space_transfers.review', TRANSFER_REVIEWS],
        ['transactions.type', spaceSchema, 'transactions.type', ALL_TRANSACTION_TYPES],
        ['transactions.status', spaceSchema, 'transactions.status', TRANSACTION_STATUSES],
        ['budgets.period', spaceSchema, 'budgets.period', BUDGET_PERIODS],
        ['automation_rules.trigger_type', spaceSchema, 'automation_rules.trigger_type', TRIGGER_TYPES],
    ]

    it.each(cases)('%s matches the frontend type', (_, schema, column, values) => {
        const fromSql = schema.get(column)
        expect(fromSql, `no CHECK ... IN (...) found for ${column}`).toBeDefined()
        expect(sorted(values)).toEqual(fromSql)
    })
})

// `sorted` folds duplicates together, so the lists are checked for them here.
describe('frontend value lists', () => {
    const lists: Record<string, readonly string[]> = {
        USER_ROLES,
        API_PERMISSION_LEVELS,
        SPACE_ROLES,
        TRANSFER_REVIEWS,
        ACCOUNT_TYPES,
        DEBT_TYPE_VALUES,
        CATEGORY_TYPES,
        RECURRING_FREQUENCIES,
        ALL_TRANSACTION_TYPES,
        TRANSACTION_STATUSES,
        BUDGET_PERIODS,
        TRIGGER_TYPES,
        CONDITION_OPERATORS,
        ACTION_TYPES,
        BACKUP_STATUSES,
        SSO_ERROR_CODES,
        MAIL_PROTOCOLS,
    }

    it.each(Object.entries(lists))('%s has no duplicates', (_, values) => {
        expect(values.filter((v, i) => values.indexOf(v) !== i)).toEqual([])
    })
})

describe('automation', () => {
    it('knows every condition operator the backend evaluates', () => {
        const body = goFunc('internal/automation/automation.go', 'func evaluateCondition(')
        expect(sorted(CONDITION_OPERATORS)).toEqual(sorted(quoted(body.match(/case [^\n]*:/g)!.join('\n'))))
    })

    it('knows every action the backend executes', () => {
        const body = goFunc('internal/automation/automation.go', 'func (s Service) executeActions(')
        expect(sorted(ACTION_TYPES)).toEqual(sorted(quoted(body.match(/case [^\n]*:/g)!.join('\n'))))
    })
})

describe('mail protocols', () => {
    it('are the ones the mailer accepts', () => {
        const source = read('internal/mail/smtp/mailer.go')
        const fromGo = [...source.matchAll(/^	[A-Za-z]+\s+= "([a-z]+)"\s+\/\//gm)].map((m) => m[1])
        expect(sorted(MAIL_PROTOCOLS)).toEqual(sorted(fromGo))
    })
})

describe('server checks', () => {
    const source = () => read('internal/httpserver/admin/checks/checks.go')

    it('are the ids the backend reports', () => {
        const fromGo = [...source().matchAll(/add\("([a-z_]+)"/g)].map((m) => m[1])
        expect(sorted(CHECK_IDS)).toEqual(sorted(fromGo))
    })

    it('have the severities the backend uses', () => {
        const fromGo = [...source().matchAll(/^	severity[A-Za-z]+\s+= "([a-z]+)"/gm)].map((m) => m[1])
        expect(sorted(CHECK_SEVERITIES)).toEqual(sorted(fromGo))
    })

    it('have a title and description', () => {
        for (const id of CHECK_IDS) {
            expect(enSettings.checks.items[id].title).toBeTruthy()
            expect(enSettings.checks.items[id].description).toBeTruthy()
        }
    })
})

describe('backup status', () => {
    it('is one of the statuses the API can return', () => {
        const body = goFunc('internal/httpserver/httpx/dto/backups.go', 'func NewBackup(')
        const fromGo = [...body.matchAll(/status\s*:?=\s*"([a-z]+)"/g)].map((m) => m[1])
        expect(sorted(BACKUP_STATUSES)).toEqual(sorted(fromGo))
    })
})

describe('SSO sign-in errors', () => {
    // Codes that can reach /auth/sso/callback?error=: everything in package sso
    // but sso.go (provider round trip, provisioning), plus the generic code the
    // handler falls back to. sso.go itself only validates admin input, whose
    // errors go to the provider form, not the callback.
    const callbackFiles = goFiles('internal/sso').filter((file) => !file.endsWith('/sso.go'))
    const emitted = callbackFiles.flatMap((file) =>
        [...read(file).matchAll(/ssoErr\(\s*"([a-z_]+)"/g)].map((m) => m[1]),
    )

    it('has a message for every code the backend sends back', () => {
        expect(emitted.length).toBeGreaterThan(10)
        expect(sorted(SSO_ERROR_CODES)).toEqual(sorted([...emitted, 'sso_error']))
    })
})

describe('space audit log', () => {
    it('has a label for every action the backend records', () => {
        const actions = goFiles('internal').flatMap((file) =>
            [...read(file).matchAll(/\.Audit\([^,]+,[^,]+,\s*"([a-z_]+)"/g)].map((m) => m[1]),
        )
        expect(actions.length).toBeGreaterThan(5)
        expect(sorted(Object.keys(enSettings.spaces.audit.actions))).toEqual(sorted(actions))
    })
})

const SPACE_DATA_PACKAGES = [
    'currencies',
    'accounts',
    'categories',
    'tags',
    'transactions',
    'imports',
    'debts',
    'reports',
    'recurring',
    'budgets',
    'automation',
    'upload',
]

describe('API routes', () => {
    it('sends every space data route through /spaces/{id}', () => {
        // Each data domain is a package under internal/httpserver that mounts its own Routes.
        const bodies = SPACE_DATA_PACKAGES.map((pkg) => {
            const file = `internal/httpserver/api/${pkg}/routes.go`
            const signature = read(file).includes('func Routes(') ? 'func Routes(' : ') Routes('
            return goFunc(file, signature)
        })
        const roots = bodies.flatMap((body) =>
            [...body.matchAll(/r\.(?:Get|Post|Put|Patch|Delete)\("\/([^/"{]+)/g)].map((m) => m[1]),
        )
        expect(roots.length).toBeGreaterThan(50)
        expect(sorted(SPACE_ROOTS)).toEqual(sorted(roots))
    })
})

describe('API token permissions', () => {
    const source = read('internal/auth/permissions/permissions.go')

    // The parts of a space and the presets come from the server (api-token-capabilities);
    // only the order of the levels, which the grid ticks by, lives here.
    it('orders the levels the backend grants', () => {
        const levels = [...source.matchAll(/^\s*(?:Read|Write|Delete)\s*= "([a-z]+)"/gm)].map((m) => m[1])
        expect(sorted(API_PERMISSION_LEVELS)).toEqual(sorted(levels))
    })

    it('has a label for every resource and preset', () => {
        const resources = /var APIResources = \[\]string\{([^}]*)\}/.exec(source)![1]
        expect(sorted(Object.keys(enSettings.api.resource))).toEqual(sorted(quoted(resources)))
        const presets = [...source.matchAll(/Preset(?:Viewer|Editor|Admin)\s*= "([a-z]+)"/g)].map((m) => m[1])
        expect(sorted(Object.keys(enSettings.api.preset).filter((k) => k !== 'custom'))).toEqual(sorted(presets))
    })
})

describe('space permissions', () => {
    it('lets the same roles write as the backend does', () => {
        const source = read('internal/spaces/space.go')
        const consts = Object.fromEntries([...source.matchAll(/(Role\w+)\s*=\s*"([a-z]+)"/g)].map((m) => [m[1], m[2]]))
        const body = /func CanWrite\([^)]*\) bool \{([^}]*)\}/.exec(source)![1]
        const writable = [...body.matchAll(/role == (Role\w+)/g)].map((m) => consts[m[1]])

        expect(writable.length).toBeGreaterThan(0)
        expect(SPACE_ROLES.filter(canWriteSpace).sort()).toEqual(writable.sort())
    })
})
