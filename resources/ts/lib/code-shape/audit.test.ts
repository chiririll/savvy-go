import { describe, it } from 'vitest'

import { isHard, RULES, scanRepo } from './scan'

/*
 * Lists code above the soft limits, worst first. Never fails: it is a worklist
 * for a cleanup session. Run it with `npm run audit`. A plain `npm test` skips it.
 */
describe.runIf(import.meta.env.MODE === 'audit')('code shape: audit', () => {
    it('prints the report', () => {
        const all = scanRepo()
        const lines: string[] = []
        for (const rule of RULES) {
            const found = all.filter((finding) => finding.rule === rule.name).sort((a, b) => b.value - a.value)
            if (found.length === 0) continue
            lines.push(`\n## ${rule.name}: ${rule.what} (soft ${rule.soft}, hard ${rule.hard}): ${found.length} found, ${found.filter(isHard).length} over hard`)
            for (const finding of found) lines.push(`${String(finding.value).padStart(5)}  ${finding.file}:${finding.line}  ${finding.name}`)
        }
        console.log(lines.length ? lines.join('\n') : '\nNothing above the soft limits.')
    })
})
