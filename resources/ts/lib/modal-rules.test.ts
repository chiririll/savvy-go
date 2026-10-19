import { readdirSync, readFileSync } from 'node:fs'
import path from 'node:path'
import ts from 'typescript'
import { describe, expect, it } from 'vitest'

/*
 * Which dialog to use. The primitives in components/ui are for the wrappers; a screen uses a wrapper:
 *
 *   - Confirm (features/spaces/settings/Confirm.tsx) or the ConfirmDialog provider: a yes/no question, no inputs.
 *   - ResponsiveDialog / FormDialog / EntityFormDialog: anything with inputs; a sheet on a phone.
 *
 * AlertDialog is for questions only: it cannot be dismissed by clicking outside, which loses a form. A raw Dialog
 * or Sheet has no phone layout of its own. A screen that breaks the rules stays in ALLOWED until it is migrated.
 */

const ROOT = path.resolve(__dirname, '..')

/** The modules whose use is limited, and the files that may use them without being on a list. */
const PRIMITIVES = {
    '@/components/ui/alert-dialog': {
        name: 'AlertDialog',
        use: 'Confirm (features/spaces/settings/Confirm.tsx)',
        wrappers: [
            'components/providers/ConfirmDialog.tsx',
            'components/providers/SessionExpiredDialog.tsx',
            'components/features/spaces/settings/Confirm.tsx',
            'components/shared/',
        ],
    },
    '@/components/ui/dialog': {
        name: 'Dialog',
        use: 'ResponsiveDialog or FormDialog (components/shared)',
        wrappers: ['components/shared/'],
    },
    '@/components/ui/sheet': {
        name: 'Sheet',
        use: 'ResponsiveDialog (components/shared)',
        wrappers: ['components/shared/', 'components/layout/'],
    },
} as const

/** JSX that takes input; inside the content of an AlertDialog or a raw Dialog it is a form in the wrong dialog. */
const INPUTS = new Set(['Input', 'Textarea', 'Select', 'Checkbox', 'Switch', 'form', 'Form', 'InputOTP', 'EmojiPicker'])

const CONTENT = new Set(['AlertDialogContent', 'DialogContent', 'SheetContent'])

/**
 * Screens that still break a rule, each with the reason and what to do about it. Remove an entry when its
 * screen is migrated: the test fails for an entry that no longer breaks any rule.
 */
const ALLOWED: Record<string, string> = {
    'components/features/admin/security/McpAccess.tsx': 'AlertDialog confirmations: move to Confirm',
    'components/features/admin/security/SigningKeys.tsx': 'AlertDialog confirmations: move to Confirm',
    'components/features/admin/spaces/DeleteSpaceButton.tsx': 'AlertDialog confirmation (type the name to delete): ResponsiveDialog',
    'components/features/api-tokens/TokenRow.tsx': 'AlertDialog confirmation: move to Confirm',
    'components/features/backups/BackupConfirmDialogs.tsx': 'AlertDialog confirmations with a password field: ResponsiveDialog',
    'components/features/categories/CategoryRow.tsx': 'AlertDialog confirmation: move to Confirm',
    'components/features/oauth/ConsentParts.tsx': 'AlertDialog confirmation: move to Confirm',
    'components/features/security/PasskeysSection.tsx': 'AlertDialog confirmation: move to Confirm',
    'components/features/spaces/settings/Invitations.tsx': 'AlertDialog confirmation: move to Confirm',
    'components/features/spaces/SyncGroupParts.tsx': 'AlertDialog confirmation: move to Confirm',
    'components/features/sso/ProviderGallery.tsx': 'raw Dialog for a picker: ResponsiveDialog',
    'components/features/transactions/SkipTransactionAlert.tsx': 'AlertDialog confirmation: move to Confirm',
    'pages/reports/components/FiltersBar.tsx': 'raw Sheet for filters: ResponsiveDialog',
}

interface Finding {
    file: string
    message: string
}

function files(dir: string): string[] {
    return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
        const full = path.join(dir, entry.name)
        if (entry.isDirectory()) return files(full)
        return entry.name.endsWith('.tsx') && !entry.name.endsWith('.test.tsx') ? [full] : []
    })
}

const tagOf = (node: ts.Node): string | null =>
    ts.isJsxElement(node)
        ? node.openingElement.tagName.getText()
        : ts.isJsxSelfClosingElement(node)
          ? node.tagName.getText()
          : null

/** The first input in the JSX under `node`, or null. */
function findInput(node: ts.Node): string | null {
    const tag = tagOf(node)
    if (tag && INPUTS.has(tag)) return tag
    let found: string | null = null
    node.forEachChild((child) => {
        found ??= findInput(child)
    })
    return found
}

export function checkSource(text: string, rel: string): Finding[] {
    const source = ts.createSourceFile(rel, text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX)
    const found: Finding[] = []
    const imported = new Map<string, string>() // local name -> primitive module

    for (const statement of source.statements) {
        if (!ts.isImportDeclaration(statement) || !ts.isStringLiteral(statement.moduleSpecifier)) continue
        const module = statement.moduleSpecifier.text as keyof typeof PRIMITIVES
        const rule = PRIMITIVES[module]
        if (!rule) continue
        const bindings = statement.importClause?.namedBindings
        if (bindings && ts.isNamedImports(bindings)) {
            for (const element of bindings.elements) imported.set(element.name.text, module)
        }
        if (!rule.wrappers.some((w) => rel.startsWith(w) || rel === w)) {
            found.push({ file: rel, message: `imports ${rule.name} from ${module}; use ${rule.use}` })
        }
    }

    const visit = (node: ts.Node) => {
        const tag = tagOf(node)
        if (tag && CONTENT.has(tag) && imported.has(tag)) {
            const input = findInput(node)
            if (input) {
                const instead = tag === 'AlertDialogContent' ? 'AlertDialog' : `a raw ${tag.replace('Content', '')}`
                const use = `ResponsiveDialog or FormDialog, not ${instead}`
                found.push({ file: rel, message: `${tag} holds <${input}>; a dialog with inputs is a ${use}` })
            }
        }
        node.forEachChild(visit)
    }
    visit(source)
    return found
}

describe('modal rules', () => {
    const all = files(ROOT)
        .filter((file) => !file.includes(`${path.sep}components${path.sep}ui${path.sep}`))
        .flatMap((file) => checkSource(readFileSync(file, 'utf8'), path.relative(ROOT, file).split(path.sep).join('/')))

    it('only lets the wrappers, and the screens on the list, use the dialog primitives', () => {
        const offenders = all
            .filter((f) => !(f.file in ALLOWED))
            .map((f) => `${f.file}: ${f.message}`)
        expect(offenders).toEqual([])
    })

    it('keeps no screen on the list that follows the rules now', () => {
        const stale = Object.keys(ALLOWED).filter((file) => !all.some((f) => f.file === file))
        expect(stale, 'remove these from ALLOWED').toEqual([])
    })

    it('names the right component for a form in an AlertDialog or a raw Dialog', () => {
        const alert = checkSource(
            `import { AlertDialog, AlertDialogContent } from '@/components/ui/alert-dialog'
             export const A = () => <AlertDialog><AlertDialogContent><Input /></AlertDialogContent></AlertDialog>`,
            'pages/x.tsx',
        )
        expect(alert.map((f) => f.message)).toEqual([
            'imports AlertDialog from @/components/ui/alert-dialog; use Confirm (features/spaces/settings/Confirm.tsx)',
            'AlertDialogContent holds <Input>; a dialog with inputs is a ResponsiveDialog or FormDialog, not AlertDialog',
        ])
        const dialog = checkSource(
            `import { Dialog, DialogContent } from '@/components/ui/dialog'
             export const A = () => <Dialog><DialogContent><form /></DialogContent></Dialog>`,
            'pages/x.tsx',
        )
        expect(dialog.map((f) => f.message).at(-1)).toBe(
            'DialogContent holds <form>; a dialog with inputs is a ResponsiveDialog or FormDialog, not a raw Dialog',
        )
    })

    it('lets the wrappers use the primitives', () => {
        const wrapper = `import { Dialog, DialogContent } from '@/components/ui/dialog'
                         export const A = () => <Dialog><DialogContent>{children}</DialogContent></Dialog>`
        expect(checkSource(wrapper, 'components/shared/ResponsiveDialog.tsx')).toEqual([])
    })
})
