import { isHard, RULES, scanRepo } from './scan.ts'

/*
 * Lists code above the soft limits, worst first. Never fails: it is a worklist
 * for a cleanup session. A script, not a test: run it with `npm run audit`.
 */
const all = scanRepo()
const lines: string[] = []
for (const rule of RULES) {
    const found = all.filter((finding) => finding.rule === rule.name).sort((a, b) => b.value - a.value)
    if (found.length === 0) continue
    const limits = `soft ${rule.soft}, hard ${rule.hard}`
    const counts = `${found.length} found, ${found.filter(isHard).length} over hard`
    lines.push(`\n## ${rule.name}: ${rule.what} (${limits}): ${counts}`)
    for (const finding of found)
        lines.push(`${String(finding.value).padStart(5)}  ${finding.file}:${finding.line}  ${finding.name}`)
}
console.log(lines.length ? lines.join('\n') : '\nNothing above the soft limits.')
