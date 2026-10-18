import { ChevronDown } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

export interface EntityFilterItem {
    id: number
    name: string
}

interface EntityFilterProps {
    label: string
    items?: EntityFilterItem[]
    selectedIds: number[]
    onToggle: (id: number) => void
    inSheet?: boolean
    formatName?: (item: EntityFilterItem) => string
    popoverClassName?: string
    hideWhenEmpty?: boolean
}

export function EntityFilter({
    label,
    items,
    selectedIds,
    onToggle,
    inSheet = false,
    formatName,
    popoverClassName,
    hideWhenEmpty = false,
}: EntityFilterProps) {
    if (hideWhenEmpty && (!items || items.length === 0)) {
        return null
    }

    const list = (
        <div className={inSheet ? 'space-y-1 max-h-40 overflow-y-auto' : 'space-y-1 max-h-64 overflow-y-auto'}>
            {items?.map((item) => (
                <div
                    key={item.id}
                    className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-muted cursor-pointer"
                    onClick={() => onToggle(item.id)}
                >
                    <Checkbox
                        checked={selectedIds.includes(item.id)}
                        className="pointer-events-none"
                    />
                    <span className="text-sm">{formatName?.(item) ?? item.name}</span>
                </div>
            ))}
        </div>
    )

    if (inSheet) {
        return (
            <div className="space-y-2">
                <Label className="text-sm font-medium">{label}</Label>
                <div className="border rounded-md p-2">{list}</div>
            </div>
        )
    }

    return (
        <Popover>
            <PopoverTrigger asChild>
                <Button variant="outline" size="sm" className="h-8">
                    {label}
                    {selectedIds.length > 0 && (
                        <Badge variant="secondary" className="ml-1 px-1.5">
                            {selectedIds.length}
                        </Badge>
                    )}
                    <ChevronDown className="ml-1 size-3" />
                </Button>
            </PopoverTrigger>
            <PopoverContent className={popoverClassName ?? 'w-56 p-2'} align="start">
                {list}
            </PopoverContent>
        </Popover>
    )
}
