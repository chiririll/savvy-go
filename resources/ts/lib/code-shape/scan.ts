import { readdirSync, readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { parseSync, visitorKeys } from 'oxc-parser'
import type { Comment, ImportDeclaration, Node } from 'oxc-parser'

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
    { name: 'funcLines', soft: 150, hard: 200, what: 'lines in a function or component' },
    // Control flow only (if, loops, switch, try). A nested function is measured on its own.
    { name: 'nesting', soft: 4, hard: 5, what: 'nested blocks in a function' },
    { name: 'complexity', soft: 15, hard: 20, what: 'branches in a function (cyclomatic complexity)' },
    { name: 'params', soft: 5, hard: 7, what: 'parameters of a function' },
    { name: 'fileLines', soft: 500, hard: 600, what: 'lines in a file' },
    { name: 'lineWidth', soft: 120, hard: 160, what: 'characters in a line' },
    // Many imports mean a file that knows too much about the rest of the code.
    { name: 'imports', soft: 15, hard: 20, what: 'imports in a file' },
    { name: 'importNames', soft: 30, hard: 40, what: 'names imported into a file' },
    { name: 'nameWords', soft: 6, hard: 8, what: 'words in a name' },
    { name: 'titleWords', soft: 12, hard: 16, what: 'words in a test title' },
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

export const REPO = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../../..')

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

type Parents = Map<Node, Node>

const isNode = (value: unknown): value is Node =>
    typeof value === 'object' && value !== null && typeof (value as { type?: unknown }).type === 'string'

/** The direct children of a node, in source order. */
function childNodes(node: Node): Node[] {
    const fields = node as unknown as Record<string, unknown>
    return (visitorKeys[node.type] ?? []).flatMap((key) => {
        const value = fields[key]
        return (Array.isArray(value) ? value : [value]).filter(isNode)
    })
}

function indexParents(root: Node): Parents {
    const parents: Parents = new Map()
    const pending = [root]
    for (let node = pending.pop(); node; node = pending.pop()) {
        for (const child of childNodes(node)) {
            parents.set(child, node)
            pending.push(child)
        }
    }
    return parents
}

export function analyzeSource(text: string, rel: string): Finding[] {
    const parsed = parseSync(rel, text, { lang: rel.endsWith('x') ? 'tsx' : 'ts', preserveParens: false })
    const failure = parsed.errors.find((error) => error.severity === 'Error')
    if (failure) throw new Error(`${rel}: ${failure.message}`)

    const program = parsed.program
    const parents = indexParents(program)
    const found: Finding[] = []

    const lineStarts = [0]
    for (let index = text.indexOf('\n'); index !== -1; index = text.indexOf('\n', index + 1)) lineStarts.push(index + 1)
    const lineOf = (offset: number) => {
        let low = 0
        let high = lineStarts.length - 1
        while (low < high) {
            const middle = (low + high + 1) >> 1
            if (lineStarts[middle] <= offset) low = middle
            else high = middle - 1
        }
        return low + 1
    }

    const add = (rule: string, at: { start: number }, name: string, value: number, ignored: Set<string>) => {
        if (value <= (RULE_BY_NAME.get(rule)?.soft ?? Infinity) || ignored.has(rule)) return
        found.push({ rule, file: rel, line: lineOf(at.start), name, value })
    }

    const checkName = (at: { start: number }, name: string, ignored: Set<string>) => {
        const words = splitWords(name)
        add('nameWords', at, name, words.length, ignored)
        add('nameJoins', at, name, words.filter((word) => word === 'And' || word === 'Or').length, ignored)
    }

    const ignoreDirectives = (node: Node, name: string) =>
        directives({ node, name, text, rel, comments: parsed.comments, parents, found, lineOf })

    // The lines of the file: the last line counts, an empty one after the final newline does not.
    const top = { start: program.body[0]?.start ?? 0 }
    add('fileLines', top, path.basename(rel), text.replace(/\n$/, '').split('\n').length, new Set())
    const imports = program.body.filter(
        (statement): statement is ImportDeclaration => statement.type === 'ImportDeclaration',
    )
    add('imports', top, path.basename(rel), imports.length, new Set())
    add(
        'importNames',
        top,
        path.basename(rel),
        imports.reduce((sum, declaration) => sum + declaration.specifiers.length, 0),
        new Set(),
    )
    const lineLimit = RULE_BY_NAME.get('lineWidth')!.soft
    text.split('\n').forEach((line, index) => {
        const width = [...line].length
        if (width > lineLimit)
            found.push({ rule: 'lineWidth', file: rel, line: index + 1, name: `line ${index + 1}`, value: width })
    })

    const visit = (node: Node) => {
        const fn = functionOf(node, parents)
        if (fn) {
            const name = functionName(fn, parents, text)
            if (name) {
                const ignored = ignoreDirectives(fn.outer, name)
                const start = declarationStart(fn.outer, parents)
                checkName(fn.key ?? fn.outer, name, ignored)
                add('params', start, name, fn.fn.params.length, ignored)
                add('funcLines', start, name, lineOf(fn.outer.end) - lineOf(start.start) + 1, ignored)
                if (fn.fn.body) {
                    const shape = measureBody(fn.fn.body)
                    add('nesting', start, name, shape.nesting, ignored)
                    add('complexity', start, name, shape.complexity, ignored)
                }
            }
        } else if (isNamedDeclaration(node) && node.id) {
            checkName(node.id, node.id.name, ignoreDirectives(node, node.id.name))
        } else if (node.type === 'CallExpression') {
            const title = testTitle(node)
            if (title !== null) add('titleWords', node, title, title.split(/\s+/).filter(Boolean).length, new Set())
        }
        childNodes(node).forEach(visit)
    }
    visit(program)
    return found
}

const NAMED_DECLARATIONS = new Set(['ClassDeclaration', 'TSInterfaceDeclaration', 'TSTypeAliasDeclaration'])

type NamedDeclaration = Extract<
    Node,
    { type: 'ClassDeclaration' | 'TSInterfaceDeclaration' | 'TSTypeAliasDeclaration' }
>

const isNamedDeclaration = (node: Node): node is NamedDeclaration => NAMED_DECLARATIONS.has(node.type)

type FunctionNode = Extract<
    Node,
    {
        type:
            | 'FunctionDeclaration'
            | 'FunctionExpression'
            | 'ArrowFunctionExpression'
            | 'TSDeclareFunction'
            | 'TSEmptyBodyFunctionExpression'
    }
>

interface FunctionInfo {
    /** The node the function is written as: a method counts with its key and modifiers. */
    outer: Node
    fn: FunctionNode
    /** The name the function declares itself, when it does. */
    key: Node | null
    computed: boolean
}

const isFunctionNode = (node: Node): node is FunctionNode =>
    node.type === 'FunctionDeclaration' ||
    node.type === 'FunctionExpression' ||
    node.type === 'ArrowFunctionExpression' ||
    node.type === 'TSDeclareFunction' ||
    node.type === 'TSEmptyBodyFunctionExpression'

/** A method or accessor holds its function as `value`; the function itself is then not a separate one. */
const isMethodValue = (node: Node, parent: Node | undefined) =>
    (parent?.type === 'MethodDefinition' || parent?.type === 'TSAbstractMethodDefinition') && parent.value === node
        ? true
        : parent?.type === 'Property' && parent.value === node && (parent.method || parent.kind !== 'init')

/**
 * The function a node declares, or null. Methods, declarations and arrows count;
 * constructors and accessors do not, but the functions inside them still do.
 */
function functionOf(node: Node, parents: Parents): FunctionInfo | null {
    switch (node.type) {
        case 'FunctionDeclaration':
        case 'TSDeclareFunction':
            return { outer: node, fn: node, key: node.id, computed: false }
        case 'FunctionExpression':
        case 'ArrowFunctionExpression':
            if (isMethodValue(node, parents.get(node))) return null
            return { outer: node, fn: node, key: node.id, computed: false }
        case 'MethodDefinition':
        case 'TSAbstractMethodDefinition':
            return node.kind === 'method'
                ? { outer: node, fn: node.value, key: node.key, computed: node.computed }
                : null
        case 'Property':
            return node.method && node.kind === 'init' && isFunctionNode(node.value)
                ? { outer: node, fn: node.value, key: node.key, computed: node.computed }
                : null
        default:
            return null
    }
}

/** `export function f` starts at the export, as the declaration it exports. */
function declarationStart(node: Node, parents: Parents): { start: number } {
    const parent = parents.get(node)
    const exported = parent?.type === 'ExportNamedDeclaration' || parent?.type === 'ExportDefaultDeclaration'
    return exported ? parent : node
}

// Wrappers an arrow can sit in while still being "the" value of a variable: memo(() => {}).
const TRANSPARENT = new Set(['CallExpression', 'ChainExpression', 'TSAsExpression'])

/** The name a function is known by, or null for a callback nobody named. */
function functionName(info: FunctionInfo, parents: Parents, text: string): string | null {
    if (info.key) return declaredName(info, parents, text)
    if (info.fn.type === 'FunctionDeclaration' || info.fn.type === 'TSDeclareFunction') return null
    return inferredName(info.fn, parents)
}

/** The name a function or method is written with; a method is prefixed with its class. */
function declaredName({ outer, key, computed }: FunctionInfo, parents: Parents, text: string): string {
    const written = text.slice(key!.start, key!.end)
    const own = computed ? `[${written}]` : written
    const klass =
        outer.type === 'MethodDefinition' || outer.type === 'TSAbstractMethodDefinition'
            ? parents.get(parents.get(outer)!)
            : undefined
    const owner = klass?.type === 'ClassDeclaration' || klass?.type === 'ClassExpression' ? klass.id?.name : undefined
    return owner ? `${owner}.${own}` : own
}

/** The name an anonymous function gets from where it sits: const Name = () => {}, also wrapped: memo(() => {}). */
function inferredName(fn: FunctionNode, parents: Parents): string | null {
    let parent = parents.get(fn)
    while (parent && TRANSPARENT.has(parent.type)) parent = parents.get(parent)
    if (parent?.type === 'VariableDeclarator' && parent.id.type === 'Identifier') return parent.id.name
    if (parent?.type === 'Property' && !parent.computed && !parent.shorthand && parent.key.type === 'Identifier')
        return parent.key.name
    if (parent?.type === 'ExportDefaultDeclaration' || parent?.type === 'TSExportAssignment') return 'default'
    return null
}

/** The title of it('...'), test('...') and describe('...'), or null for any other call. */
function testTitle(call: Extract<Node, { type: 'CallExpression' }>): string | null {
    if (call.callee.type !== 'Identifier' || !TEST_FUNCTIONS.has(call.callee.name)) return null
    const first = call.arguments[0]
    if (first?.type === 'Literal' && typeof first.value === 'string') return first.value
    if (first?.type === 'TemplateLiteral' && first.expressions.length === 0) return first.quasis[0].value.cooked ?? null
    return null
}

const LOOPS = new Set(['ForStatement', 'ForInStatement', 'ForOfStatement', 'WhileStatement', 'DoWhileStatement'])
const LOGICAL_BRANCHES = new Set(['&&', '||', '??'])

const BLOCK_OPENERS = new Set(['IfStatement', 'SwitchStatement', 'TryStatement', ...LOOPS])

const opensBlock = (node: Node) => BLOCK_OPENERS.has(node.type)

/** The else-if of an if statement continues its chain and adds no level. */
const elseIf = (node: Node) =>
    node.type === 'IfStatement' && node.alternate?.type === 'IfStatement' ? node.alternate : undefined

function addsBranch(node: Node): boolean {
    switch (node.type) {
        case 'IfStatement':
        case 'ConditionalExpression':
        case 'CatchClause':
            return true
        case 'SwitchCase':
            return node.test !== null
        case 'LogicalExpression':
            return LOGICAL_BRANCHES.has(node.operator)
        default:
            return LOOPS.has(node.type)
    }
}

/** Nesting depth and cyclomatic complexity of a body, without looking inside nested functions. */
function measureBody(body: Node): { nesting: number; complexity: number } {
    let deepest = 0
    let complexity = 1
    const walk = (node: Node, depth: number) => {
        if (isFunctionNode(node) || node.type === 'ClassDeclaration' || node.type === 'ClassExpression') return
        let next = depth
        if (opensBlock(node)) {
            next = depth + 1
            deepest = Math.max(deepest, next)
        }
        if (addsBranch(node)) complexity++
        const chained = elseIf(node)
        for (const child of childNodes(node)) walk(child, child === chained ? depth : next)
    }
    for (const child of childNodes(body)) walk(child, 0)
    return { nesting: deepest, complexity }
}

interface DirectiveContext {
    node: Node
    name: string
    text: string
    rel: string
    comments: Comment[]
    parents: Parents
    found: Finding[]
    lineOf: (offset: number) => number
}

const STATEMENT_CONTAINERS = new Set(['Program', 'BlockStatement', 'ClassBody', 'TSModuleBlock', 'StaticBlock'])

/** The comments right above a position that start on their own line, nearest last. */
function leadingComments(comments: Comment[], text: string, start: number): Comment[] {
    const above: Comment[] = []
    let cursor = start
    for (let index = comments.length - 1; index >= 0; index--) {
        const comment = comments[index]
        if (comment.end > start) continue
        if (text.slice(comment.end, cursor).trim() !== '') break
        above.unshift(comment)
        cursor = comment.start
    }

    // A comment on the line of the previous token belongs to that token, not to what follows.
    let previousEnd = cursor
    while (previousEnd > 0 && /\s/.test(text[previousEnd - 1])) previousEnd--
    if (previousEnd === 0) return above
    let newlineSeen = false
    return above.filter((comment) => {
        newlineSeen ||= text.slice(previousEnd, comment.start).includes('\n')
        previousEnd = comment.end
        return newlineSeen
    })
}

/**
 * Rules switched off by `// archtest:ignore <rule> <reason>` above a function
 * or type. A directive without a reason is reported and does not switch anything off.
 */
function directives({ node, name, text, rel, comments, parents, found, lineOf }: DirectiveContext): Set<string> {
    const ignored = new Set<string>()
    let owner = node
    // const Name = () => {} carries its comment on the statement, not the arrow.
    for (let parent = parents.get(owner); parent && !STATEMENT_CONTAINERS.has(parent.type); parent = parents.get(owner))
        owner = parent
    for (const comment of leadingComments(comments, text, owner.start)) {
        const match = /archtest:ignore\s+(\S+)(?:\s+(\S.*))?/.exec(text.slice(comment.start, comment.end))
        if (!match) continue
        if (match[2]) {
            ignored.add(match[1])
        } else {
            found.push({ rule: 'ignoreWithoutReason', file: rel, line: lineOf(comment.start), name, value: 1 })
        }
    }
    return ignored
}

function sourceFiles(dir: string): string[] {
    return readdirSync(path.join(REPO, dir), { withFileTypes: true }).flatMap((entry) => {
        const rel = `${dir}/${entry.name}`
        if (entry.isDirectory())
            return SKIPPED_DIRS.some((skipped) => `${rel}/`.startsWith(skipped)) ? [] : sourceFiles(rel)
        return /\.tsx?$/.test(entry.name) && !entry.name.endsWith('.d.ts') ? [rel] : []
    })
}

/** Measures every hand-written file under resources/ts. */
export function scanRepo(): Finding[] {
    return sourceFiles('resources/ts')
        .flatMap((rel) => analyzeSource(readFileSync(path.join(REPO, rel), 'utf8').replace(/\r\n/g, '\n'), rel))
        .sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line)
}
