import { readdirSync, readFileSync } from 'node:fs'
import path from 'node:path'
import { describe, expect, it } from 'vitest'

/*
 * Which dialog to use. The primitives in components/ui are for the wrappers; a screen uses a wrapper:
 *
 *   - Confirm (components/shared/dialogs/Confirm.tsx): a yes/no question, no inputs.
 *   - ResponsiveDialog / FormDialog / EntityFormDialog: anything with inputs; a sheet on a phone.
 *
 * A pattern with no wrapper gets one in components/shared/dialogs, with a parameter for what differs; a screen
 * does not work around it with a primitive.
 */

const ROOT = path.resolve(__dirname, '..')

/** The modules whose use is limited, and the files that may import them. */
const PRIMITIVES: Record<string, { use: string; wrappers: string[] }> = {
    '@/components/ui/alert-dialog': {
        use: 'Confirm',
        wrappers: ['components/providers/SessionExpiredDialog.tsx', 'components/shared/'],
    },
    '@/components/ui/dialog': { use: 'ResponsiveDialog or FormDialog', wrappers: ['components/shared/'] },
    '@/components/ui/sheet': { use: 'ResponsiveDialog', wrappers: ['components/shared/', 'components/layout/'] },
}

const IMPORT = /from\s+['"](@\/components\/ui\/[\w-]+)['"]/g

function files(dir: string): string[] {
    return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
        const full = path.join(dir, entry.name)
        if (entry.isDirectory()) return files(full)
        return /\.tsx?$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name) ? [full] : []
    })
}

/** The problems with the imports of one file, as messages. */
export function checkSource(text: string, rel: string): string[] {
    const found: string[] = []
    for (const [, module] of text.matchAll(IMPORT)) {
        const rule = PRIMITIVES[module]
        if (rule && !rule.wrappers.some((w) => rel === w || rel.startsWith(w))) {
            found.push(`${rel} imports ${module}; use ${rule.use}`)
        }
    }
    return found
}

describe('modal rules', () => {
    it('only lets the wrappers import the dialog primitives', () => {
        const offenders = files(ROOT)
            .map((file) => path.relative(ROOT, file).split(path.sep).join('/'))
            .filter((rel) => !rel.startsWith('components/ui/'))
            .flatMap((rel) => checkSource(readFileSync(path.join(ROOT, rel), 'utf8'), rel))
        expect(offenders).toEqual([])
    })

    it('names the wrapper to use instead', () => {
        const source = `import { AlertDialog } from '@/components/ui/alert-dialog'`
        expect(checkSource(source, 'pages/x.tsx')).toEqual(['pages/x.tsx imports @/components/ui/alert-dialog; use Confirm'])
    })

    it('lets the wrappers import them', () => {
        const source = `import { Dialog } from '@/components/ui/dialog'`
        expect(checkSource(source, 'components/shared/dialogs/ResponsiveDialog.tsx')).toEqual([])
    })
})
