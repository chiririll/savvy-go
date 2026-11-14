interface TooltipRow {
    color: string
    label: string
    /** Already formatted amount. */
    amount: string
    /** Not the line under the cursor: shown faded. */
    dimmed?: boolean
}

/** One coloured dot, series name and amount line of a chart tooltip. */
export function tooltipRow({ color, label, amount, dimmed = false }: TooltipRow): string {
    return `<div class="flex items-center gap-2" style="opacity:${dimmed ? 0.4 : 1}">
                        <span class="w-2 h-2 rounded-full" style="background:${color}"></span>
                        <span>${label}: <strong>${amount}</strong></span>
                    </div>`
}

/** Whether a row for the series at `index` should fade while `hoveredIndex` is under the cursor. */
export function isDimmed(hoveredIndex: number | null, index: number): boolean {
    return hoveredIndex !== null && hoveredIndex !== index
}
