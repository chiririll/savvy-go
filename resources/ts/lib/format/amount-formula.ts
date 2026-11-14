import { sanitizeAmount } from './amount-input'

/**
 * Formulas in an amount field: numbers with `+ - * /` and parentheses, evaluated when the field loses focus. The
 * maths is exact (fractions of BigInts), never float; only the result is rounded, to the decimals of the currency.
 */

export type FormulaResult =
    | { kind: 'plain' }
    | { kind: 'value'; value: string }
    | { kind: 'error' }

const OPERATOR = /[+*/()=]|\d\s*-|[.,)]\s*-/

/** Whether the text is more than a number: it has an operator, a parenthesis or a leading `=`. */
export function isFormula(text: string): boolean {
    return OPERATOR.test(text)
}

/**
 * What an amount field lets through while typing: a plain amount as `sanitizeAmount` has it, a formula with only
 * its own characters (the decimals are cut when it is evaluated).
 */
export function sanitizeAmountInput(raw: string, decimals: number): string {
    if (!isFormula(raw)) {
        return sanitizeAmount(raw, decimals)
    }
    return raw.replace(/[^\d.,+\-*/()=\s]/g, '')
}

interface Fraction {
    num: bigint
    den: bigint
}

const gcd = (a: bigint, b: bigint): bigint => {
    let x = a < 0n ? -a : a
    let y = b < 0n ? -b : b
    while (y) {
        ;[x, y] = [y, x % y]
    }
    return x
}

const frac = (num: bigint, den: bigint): Fraction => {
    const g = gcd(num, den) || 1n
    return den < 0n ? { num: -num / g, den: -den / g } : { num: num / g, den: den / g }
}

class Failure extends Error {}

/** Recursive descent over the characters: expr = term {(+|-) term}; term = factor {(*|/) factor}. */
class Parser {
    private pos = 0
    constructor(private readonly text: string) {}

    parse(): Fraction {
        const value = this.expr()
        this.skip()
        if (this.pos !== this.text.length) throw new Failure()
        return value
    }

    private skip(): void {
        while (this.text[this.pos] === ' ') this.pos++
    }

    private peek(): string | undefined {
        this.skip()
        return this.text[this.pos]
    }

    private expr(): Fraction {
        let left = this.term()
        for (let op = this.peek(); op === '+' || op === '-'; op = this.peek()) {
            this.pos++
            const right = this.term()
            const sign = op === '+' ? 1n : -1n
            left = frac(left.num * right.den + sign * right.num * left.den, left.den * right.den)
        }
        return left
    }

    private term(): Fraction {
        let left = this.factor()
        for (let op = this.peek(); op === '*' || op === '/'; op = this.peek()) {
            this.pos++
            const right = this.factor()
            if (op === '*') {
                left = frac(left.num * right.num, left.den * right.den)
            } else {
                if (right.num === 0n) throw new Failure()
                left = frac(left.num * right.den, left.den * right.num)
            }
        }
        return left
    }

    private factor(): Fraction {
        const c = this.peek()
        if (c === '-' || c === '+') {
            this.pos++
            const inner = this.factor()
            return c === '-' ? { num: -inner.num, den: inner.den } : inner
        }
        if (c === '(') {
            this.pos++
            const inner = this.expr()
            if (this.peek() !== ')') throw new Failure()
            this.pos++
            return inner
        }
        return this.number()
    }

    private number(): Fraction {
        this.skip()
        const match = /^(\d*)(?:[.,](\d*))?/.exec(this.text.slice(this.pos))
        const whole = match?.[1] ?? ''
        const fraction = match?.[2] ?? ''
        if (!match || (whole === '' && fraction === '')) throw new Failure()
        this.pos += match[0].length
        return frac(BigInt(`${whole || '0'}${fraction}`), 10n ** BigInt(fraction.length))
    }
}

/**
 * Evaluates the text when it is a formula. A plain number is `plain` (the caller keeps it); a formula that does not
 * parse, divides by zero or comes out negative is an `error`, and the caller keeps the text. The result is rounded
 * half up to `decimals`, as `roundToCurrency` rounds, and written with exactly that many decimals.
 */
export function evaluateAmount(text: string, decimals: number): FormulaResult {
    if (!isFormula(text)) {
        return { kind: 'plain' }
    }
    try {
        const body = text.trim().replace(/^=/, '')
        const { num, den } = new Parser(body).parse()
        if (num < 0n) return { kind: 'error' }
        const scale = 10n ** BigInt(Math.max(decimals, 0))
        const units = (2n * num * scale + den) / (2n * den)
        const digits = units.toString().padStart(decimals + 1, '0')
        const cut = digits.length - decimals
        return { kind: 'value', value: decimals > 0 ? `${digits.slice(0, cut)}.${digits.slice(cut)}` : digits }
    } catch (error) {
        if (error instanceof Failure) return { kind: 'error' }
        throw error
    }
}
