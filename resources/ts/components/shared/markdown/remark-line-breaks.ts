interface MdNode {
    type: string
    value?: string
    children?: MdNode[]
}

/** Splits the text on its line feeds, with a hard break in between. */
function split(node: MdNode): MdNode[] {
    const parts = (node.value ?? '').split('\n')
    return parts.flatMap((part, i) => {
        const out: MdNode[] = i > 0 ? [{ type: 'break' }] : []
        if (part !== '') out.push({ ...node, value: part })
        return out
    })
}

function walk(node: MdNode) {
    if (!node.children) return
    node.children = node.children.flatMap((child) => {
        if (child.type === 'text' && child.value?.includes('\n')) return split(child)
        walk(child)
        return [child]
    })
}

/**
 * A line feed in the editor is a line break in the preview, as in a chat or a note, not a space as in CommonMark.
 * Code is untouched: its text is of other node types.
 */
export function remarkLineBreaks() {
    return (tree: MdNode) => walk(tree)
}
