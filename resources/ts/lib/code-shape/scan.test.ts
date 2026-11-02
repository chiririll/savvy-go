import { describe, expect, it } from 'vitest'

import { analyzeSource, splitWords } from './scan'
import type { Finding } from './scan'

const scan = (code: string, rel = 'sample.ts'): Finding[] => analyzeSource(code, rel)
const only = (findings: Finding[], rule: string) => findings.filter((finding) => finding.rule === rule)
const repeat = (count: number, line: (index: number) => string) =>
    Array.from({ length: count }, (_, index) => line(index)).join('\n')

describe('splitWords', () => {
    it('cuts camel case, constants and digits', () => {
        expect(splitWords('useSpaceInvitations')).toEqual(['use', 'Space', 'Invitations'])
        expect(splitWords('MAX_ROWS')).toEqual(['MAX', 'ROWS'])
        expect(splitWords('parseHTMLString')).toEqual(['parse', 'HTML', 'String'])
    })
})

describe('analyzeSource: functions', () => {
    it('counts parameters of a declaration', () => {
        const found = only(scan('function build(a, b, c, d, e, f) {}'), 'params')
        expect(found).toEqual([{ rule: 'params', file: 'sample.ts', line: 1, name: 'build', value: 6 }])
    })

    it('names a method after its class', () => {
        const found = only(scan('class Store {\n    save(a, b, c, d, e, f) {}\n}'), 'params')
        expect(found).toMatchObject([{ name: 'Store.save', line: 2, value: 6 }])
    })

    it('names an object method after its key', () => {
        const found = only(scan('const api = {\n    save(a, b, c, d, e, f) {},\n}'), 'params')
        expect(found).toMatchObject([{ name: 'save', line: 2 }])
    })

    it('names an arrow after its variable, also when wrapped in calls', () => {
        const code = [
            'const plain = (a, b, c, d, e, f) => {}',
            'const wrapped = memo((a, b, c, d, e, f) => {})',
            'const cast = ((a, b, c, d, e, f) => {}) as Handler',
        ].join('\n')
        expect(only(scan(code), 'params').map((finding) => finding.name)).toEqual(['plain', 'wrapped', 'cast'])
    })

    it('names an arrow after its property', () => {
        const found = only(scan('const map = {\n    handler: (a, b, c, d, e, f) => {},\n}'), 'params')
        expect(found).toMatchObject([{ name: 'handler', line: 2 }])
    })

    it('names a default-exported arrow "default" and skips an anonymous default function', () => {
        expect(only(scan('export default (a, b, c, d, e, f) => {}'), 'params')).toMatchObject([{ name: 'default' }])
        expect(only(scan('export default function (a, b, c, d, e, f) {}'), 'params')).toEqual([])
    })

    it('skips callbacks nobody named, constructors and accessors', () => {
        const code = [
            'run((a, b, c, d, e, f) => {})',
            'class A {',
            '    constructor(a, b, c, d, e, f) {}',
            '    get value() { return 1 }',
            '}',
        ].join('\n')
        expect(only(scan(code), 'params')).toEqual([])
    })

    it('starts an exported declaration at the export keyword', () => {
        const code = '\n\nexport function build(a, b, c, d, e, f) {}'
        expect(only(scan(code), 'params')).toMatchObject([{ line: 3 }])
    })

    it('measures function length from the first to the last line', () => {
        const body = repeat(150, () => '    work()')
        const found = only(scan(`function long() {\n${body}\n}`), 'funcLines')
        expect(found).toMatchObject([{ name: 'long', line: 1, value: 152 }])
    })

    it('counts branches: if, loops, case, catch, ?:, &&, || and ??', () => {
        const body = repeat(16, () => '    if (a && b) work()')
        const found = only(scan(`function busy() {\n${body}\n}`), 'complexity')
        expect(found).toMatchObject([{ name: 'busy', value: 33 }])
    })

    it('counts case clauses but not default', () => {
        const cases = repeat(16, (index) => `        case ${index}: break`)
        const code = `function pick(x) {\n    switch (x) {\n${cases}\n        default: break\n    }\n}`
        expect(only(scan(code), 'complexity')).toMatchObject([{ value: 17 }])
    })

    it('keeps a nested function out of the outer complexity', () => {
        const inner = repeat(16, () => '        if (a) work()')
        const code = `function outer() {\n    const inner = () => {\n${inner}\n    }\n}`
        const found = only(scan(code), 'complexity')
        expect(found.map((finding) => [finding.name, finding.value])).toEqual([['inner', 17]])
    })

    it('measures nesting of control flow', () => {
        const code = [
            'function deep() {',
            '    if (a) { for (const x of y) { while (b) { try { if (c) { work() } } finally {} } } }',
            '}',
        ].join('\n')
        expect(only(scan(code), 'nesting')).toMatchObject([{ name: 'deep', value: 5 }])
    })

    it('does not nest an else-if chain', () => {
        const chain = Array.from({ length: 8 }, () => 'if (a) { work() }').join(' else ')
        expect(only(scan(`function flat() { ${chain} }`), 'nesting')).toEqual([])
    })
})

describe('analyzeSource: files', () => {
    it('counts imports and the names they bring in', () => {
        const imports = repeat(16, (index) => `import { a${index}, b${index} } from 'm${index}'`)
        const code = `import 'side-effect'\nimport def, * as ns from 'x'\n${imports}\n`
        const found = scan(code)
        expect(only(found, 'imports')).toMatchObject([{ value: 18 }])
        expect(only(found, 'importNames')).toMatchObject([{ value: 16 * 2 + 2 }])
    })

    it('counts the lines of a file, not a trailing empty one', () => {
        const code = repeat(501, () => 'work()') + '\n'
        expect(only(scan(code), 'fileLines')).toMatchObject([{ value: 501 }])
    })

    it('reports a line wider than the limit', () => {
        const code = `const x = 1\nconst y = '${'x'.repeat(130)}'\n`
        expect(only(scan(code), 'lineWidth')).toMatchObject([{ line: 2, name: 'line 2' }])
    })

    it('parses tsx', () => {
        const code = 'export function View(a, b, c, d, e, f) {\n    return <div>{a}</div>\n}'
        expect(only(scan(code, 'View.tsx'), 'params')).toMatchObject([{ name: 'View' }])
    })
})

describe('analyzeSource: names and titles', () => {
    it('counts words in functions, classes, interfaces and type aliases', () => {
        const code = [
            'function oneTwoThreeFourFiveSixSeven() {}',
            'class OneTwoThreeFourFiveSixSeven {}',
            'interface OneTwoThreeFourFiveSixSeven {}',
            'type OneTwoThreeFourFiveSixSeven = string',
        ].join('\n')
        const found = only(scan(code), 'nameWords')
        expect(found.map((finding) => [finding.line, finding.value])).toEqual([
            [1, 7],
            [2, 7],
            [3, 7],
            [4, 7],
        ])
    })

    it('flags And and Or joining two ideas', () => {
        const found = only(scan('function saveAndClose() {}'), 'nameJoins')
        expect(found).toMatchObject([{ name: 'saveAndClose', value: 1 }])
    })

    it('counts words in test titles, also in template literals', () => {
        const title = 'a b c d e f g h i j k l m'
        const code = `it('${title}', () => {})\ndescribe(\`${title}\`, () => {})\nit.each([])('${title}', () => {})`
        const found = only(scan(code), 'titleWords')
        expect(found.map((finding) => [finding.line, finding.value])).toEqual([
            [1, 13],
            [2, 13],
        ])
    })
})

describe('analyzeSource: archtest:ignore', () => {
    const wide = 'function build(a, b, c, d, e, f) {}'

    it('switches a rule off for the function below it', () => {
        const code = `// archtest:ignore params wide on purpose\n${wide}`
        expect(only(scan(code), 'params')).toEqual([])
    })

    it('reaches a declaration through its export and variable statement', () => {
        const code = '// archtest:ignore params wide on purpose\nexport const build = (a, b, c, d, e, f) => {}'
        expect(only(scan(code), 'params')).toEqual([])
    })

    it('only switches off the named rule', () => {
        const code = `// archtest:ignore funcLines reason\n${wide}`
        expect(only(scan(code), 'params')).toHaveLength(1)
    })

    it('reports a directive without a reason and keeps the rule on', () => {
        const code = `// archtest:ignore params\n${wide}`
        const found = scan(code)
        expect(only(found, 'ignoreWithoutReason')).toMatchObject([{ line: 1, name: 'build' }])
        expect(only(found, 'params')).toHaveLength(1)
    })

    it('does not take a comment on the line of the previous statement', () => {
        const code = `const a = 1 // archtest:ignore params reason\n${wide}`
        expect(only(scan(code), 'params')).toHaveLength(1)
    })

    it('does not reach across a blank statement', () => {
        const code = `// archtest:ignore params reason\nconst a = 1\n${wide}`
        expect(only(scan(code), 'params')).toHaveLength(1)
    })
})
