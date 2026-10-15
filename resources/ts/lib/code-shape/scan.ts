import { readdirSync, readFileSync } from 'node:fs'
import path from 'node:path'
import ts from 'typescript'

/*
 * Measures how hard the frontend code is to read. The Go side does the same in
 * internal/archtest; the rules match.
 *
 * A value above `soft` is worth a look in the audit report (npm run audit); a
 * value above `hard` fails npm test.
 */

export interface Rule {
    name: string
    soft: number
    hard: number
    what: string
}

export const RULES: Rule[] = [
    // Lines include nested functions, so a component counts its handlers and JSX.
    { name: 'funcLines', soft: 150, hard: 500, what: 'lines in a function or component' },
    // Control flow only (if, loops, switch, try). A nested function is measured on its own.
    { name: 'nesting', soft: 4, hard: 8, what: 'nested blocks in a function' },
    { name: 'complexity', soft: 15, hard: 40, what: 'branches in a function (cyclomatic complexity)' },
    { name: 'params', soft: 5, hard: 10, what: 'parameters of a function' },
    { name: 'fileLines', soft: 500, hard: 1500, what: 'lines in a file' },
    { name: 'lineWidth', soft: 120, hard: 300, what: 'characters in a line' },
    // Many imports mean a file that knows too much about the rest of the code.
    { name: 'imports', soft: 15, hard: 40, what: 'imports in a file' },
    { name: 'importNames', soft: 30, hard: 80, what: 'names imported into a file' },
    { name: 'nameWords', soft: 6, hard: 10, what: 'words in a name' },
    { name: 'titleWords', soft: 12, hard: 18, what: 'words in a test title' },
    // A name that joins two ideas with And/Or usually does two jobs. Soft only:
    // a hard limit would punish legitimate names like validateOrFail.
    { name: 'nameJoins', soft: 0, hard: Number.MAX_SAFE_INTEGER, what: 'And/Or in a name' },
    { name: 'ignoreWithoutReason', soft: 0, hard: 0, what: 'archtest:ignore without a reason' },
]

const RULE_BY_NAME = new Map(RULES.map((rule) => [rule.name, rule]))

export interface Finding {
    rule: string
    /** Relative to the repository root, with forward slashes. */
    file: string
    line: number
    name: string
    value: number
}

export const isHard = (finding: Finding) => finding.value > (RULE_BY_NAME.get(finding.rule)?.hard ?? Infinity)

export const REPO = path.resolve(__dirname, '../../../..')

// Shadcn primitives are copied in, not written here.
const SKIPPED_DIRS = ['resources/ts/components/ui/']

/** Cuts a name into words: "useSpaceInvitations" is use, Space, Invitations; "MAX_ROWS" is MAX, ROWS. */
export function splitWords(name: string): string[] {
    return name
        .replace(/([a-z\d])([A-Z])/g, '$1 $2')
        .replace(/([A-Z]+)([A-Z][a-z])/g, '$1 $2')
        .replace(/([A-Za-z])(\d)/g, '$1 $2')
        .split(/[\s_$-]+/)
        .filter(Boolean)
}

const TEST_FUNCTIONS = new Set(['it', 'test', 'describe'])

export function analyzeSource(text: string, rel: string): Finding[] {
    const source = ts.createSourceFile(rel, text, ts.ScriptTarget.Latest, true, rel.endsWith('x') ? ts.ScriptKind.TSX : ts.ScriptKind.TS)
    const found: Finding[] = []

    const lineOf = (pos: number) => source.getLineAndCharacterOfPosition(pos).line + 1
    const add = (rule: string, node: ts.Node, name: string, value: number, ignored: Set<string>) => {
        if (value <= (RULE_BY_NAME.get(rule)?.soft ?? Infinity) || ignored.has(rule)) return
        found.push({ rule, file: rel, line: lineOf(node.getStart(source)), name, value })
    }

    const checkName = (node: ts.Node, name: string, ignored: Set<string>) => {
        const words = splitWords(name)
        add('nameWords', node, name, words.length, ignored)
        add('nameJoins', node, name, words.filter((word) => word === 'And' || word === 'Or').length, ignored)
    }

    // The lines of the file: the last line counts, an empty one after the final newline does not.
    add('fileLines', source, path.basename(rel), text.replace(/\n$/, '').split('\n').length, new Set())
    const imports = source.statements.filter(ts.isImportDeclaration)
    add('imports', source, path.basename(rel), imports.length, new Set())
    add('importNames', source, path.basename(rel), imports.reduce((sum, declaration) => sum + importedNames(declaration), 0), new Set())
    const lineLimit = RULE_BY_NAME.get('lineWidth')!.soft
    text.split('\n').forEach((line, index) => {
        const width = [...line].length
        if (width > lineLimit) found.push({ rule: 'lineWidth', file: rel, line: index + 1, name: `line ${index + 1}`, value: width })
    })

    const visit = (node: ts.Node) => {
        if (isFunctionLike(node)) {
            const name = functionName(node)
            if (name) {
                const ignored = directives(node, source, found, rel, name)
                checkName(node.name ?? node, name, ignored)
                add('params', node, name, node.parameters.length, ignored)
                add('funcLines', node, name, lineOf(node.end) - lineOf(node.getStart(source)) + 1, ignored)
                if (node.body) {
                    const shape = measureBody(node.body)
                    add('nesting', node, name, shape.nesting, ignored)
                    add('complexity', node, name, shape.complexity, ignored)
                }
            }
        } else if (ts.isInterfaceDeclaration(node) || ts.isTypeAliasDeclaration(node) || ts.isClassDeclaration(node)) {
            if (node.name) checkName(node.name, node.name.text, directives(node, source, found, rel, node.name.text))
        } else if (ts.isCallExpression(node)) {
            const title = testTitle(node)
            if (title !== null) add('titleWords', node, title, title.split(/\s+/).filter(Boolean).length, new Set())
        }
        ts.forEachChild(node, visit)
    }
    visit(source)
    return found
}

/** Names an import brings in: the default, the namespace and each named binding. */
function importedNames(declaration: ts.ImportDeclaration): number {
    const clause = declaration.importClause
    if (!clause) return 0
    const bindings = clause.namedBindings
    return (clause.name ? 1 : 0) + (bindings ? (ts.isNamedImports(bindings) ? bindings.elements.length : 1) : 0)
}

type FunctionLike = ts.FunctionDeclaration | ts.MethodDeclaration | ts.ArrowFunction | ts.FunctionExpression

function isFunctionLike(node: ts.Node): node is FunctionLike {
    return ts.isFunctionDeclaration(node) || ts.isMethodDeclaration(node) || ts.isArrowFunction(node) || ts.isFunctionExpression(node)
}

/** The name a function is known by, or null for a callback nobody named. */
function functionName(node: FunctionLike): string | null {
    if (node.name) {
        const own = node.name.getText()
        const owner = ts.isMethodDeclaration(node) && ts.isClassLike(node.parent) ? node.parent.name?.text : undefined
        return owner ? `${owner}.${own}` : own
    }
    // const Name = () => {}, also when wrapped: memo(() => {}), useCallback(() => {}, []).
    let parent: ts.Node = node.parent
    while (ts.isParenthesizedExpression(parent) || ts.isCallExpression(parent) || ts.isAsExpression(parent)) parent = parent.parent
    if (ts.isVariableDeclaration(parent) && ts.isIdentifier(parent.name)) return parent.name.text
    if (ts.isPropertyAssignment(parent) && ts.isIdentifier(parent.name)) return parent.name.text
    if (ts.isExportAssignment(parent)) return 'default'
    return null
}

/** The title of it('...'), test('...') and describe('...'), or null for any other call. */
function testTitle(call: ts.CallExpression): string | null {
    if (!ts.isIdentifier(call.expression) || !TEST_FUNCTIONS.has(call.expression.text)) return null
    const first = call.arguments[0]
    return first && ts.isStringLiteralLike(first) ? first.text : null
}

/** Nesting depth and cyclomatic complexity of a body, without looking inside nested functions. */
function measureBody(body: ts.Node): { nesting: number; complexity: number } {
    let deepest = 0
    let complexity = 1
    const walk = (node: ts.Node, depth: number) => {
        if (isFunctionLike(node) || ts.isClassLike(node)) return
        let next = depth
        if (opensBlock(node)) {
            next = depth + 1
            deepest = Math.max(deepest, next)
        }
        if (addsBranch(node)) complexity++
        ts.forEachChild(node, (child) => walk(child, child === elseIf(node) ? depth : next))
    }
    ts.forEachChild(body, (child) => walk(child, 0))
    return { nesting: deepest, complexity }
}

const opensBlock = (node: ts.Node) =>
    ts.isIfStatement(node) ||
    ts.isForStatement(node) ||
    ts.isForInStatement(node) ||
    ts.isForOfStatement(node) ||
    ts.isWhileStatement(node) ||
    ts.isDoStatement(node) ||
    ts.isSwitchStatement(node) ||
    ts.isTryStatement(node)

/** The else-if of an if statement continues its chain and adds no level. */
const elseIf = (node: ts.Node) => (ts.isIfStatement(node) && node.elseStatement && ts.isIfStatement(node.elseStatement) ? node.elseStatement : undefined)

function addsBranch(node: ts.Node): boolean {
    if (
        ts.isIfStatement(node) ||
        ts.isForStatement(node) ||
        ts.isForInStatement(node) ||
        ts.isForOfStatement(node) ||
        ts.isWhileStatement(node) ||
        ts.isDoStatement(node) ||
        ts.isCaseClause(node) ||
        ts.isCatchClause(node) ||
        ts.isConditionalExpression(node)
    ) {
        return true
    }
    if (ts.isBinaryExpression(node)) {
        const kind = node.operatorToken.kind
        return kind === ts.SyntaxKind.AmpersandAmpersandToken || kind === ts.SyntaxKind.BarBarToken || kind === ts.SyntaxKind.QuestionQuestionToken
    }
    return false
}

/**
 * Rules switched off by `// archtest:ignore <rule> <reason>` above a function
 * or type. A directive without a reason is reported and does not switch anything off.
 */
function directives(node: ts.Node, source: ts.SourceFile, found: Finding[], rel: string, name: string): Set<string> {
    const ignored = new Set<string>()
    let owner: ts.Node = node
    // const Name = () => {} carries its comment on the statement, not the arrow.
    while (owner.parent && !ts.isSourceFile(owner.parent) && !ts.isBlock(owner.parent) && !ts.isClassLike(owner.parent) && !ts.isModuleBlock(owner.parent)) {
        owner = owner.parent
    }
    for (const range of ts.getLeadingCommentRanges(source.text, owner.getFullStart()) ?? []) {
        const match = /archtest:ignore\s+(\S+)(?:\s+(\S.*))?/.exec(source.text.slice(range.pos, range.end))
        if (!match) continue
        if (match[2]) {
            ignored.add(match[1])
        } else {
            found.push({ rule: 'ignoreWithoutReason', file: rel, line: source.getLineAndCharacterOfPosition(range.pos).line + 1, name, value: 1 })
        }
    }
    return ignored
}

function sourceFiles(dir: string): string[] {
    return readdirSync(path.join(REPO, dir), { withFileTypes: true }).flatMap((entry) => {
        const rel = `${dir}/${entry.name}`
        if (entry.isDirectory()) return SKIPPED_DIRS.some((skipped) => `${rel}/`.startsWith(skipped)) ? [] : sourceFiles(rel)
        return /\.tsx?$/.test(entry.name) && !entry.name.endsWith('.d.ts') ? [rel] : []
    })
}

/** Measures every hand-written file under resources/ts. */
export function scanRepo(): Finding[] {
    return sourceFiles('resources/ts')
        .flatMap((rel) => analyzeSource(readFileSync(path.join(REPO, rel), 'utf8').replace(/\r\n/g, '\n'), rel))
        .sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line)
}
