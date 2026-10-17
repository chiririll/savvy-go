import { describe, expect, it } from 'vitest'

import { isHard, RULES, scanRepo } from './scan'

describe('code shape: hard limits', () => {
    it('keeps all code within them', () => {
        const offenders = scanRepo()
            .filter(isHard)
            .map((finding) => {
                const rule = RULES.find((r) => r.name === finding.rule)!
                return `${finding.file}:${finding.line} ${finding.name}: ${finding.value} ${rule.what} (hard limit ${rule.hard}). Split it up, or opt out with // archtest:ignore ${rule.name} <reason>`
            })
        expect(offenders).toEqual([])
    }, 30_000)
})
