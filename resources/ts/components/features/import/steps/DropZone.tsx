import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FileText, Upload, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface Props {
    onFileSelect: (file: File) => void
    disabled: boolean
}

const isCsv = (file: File) => file.type === 'text/csv' || file.name.endsWith('.csv')

export function DropZone({ onFileSelect, disabled }: Props) {
    const { t } = useTranslation('settings')
    const [dragActive, setDragActive] = useState(false)
    const [selectedFile, setSelectedFile] = useState<File | null>(null)

    const select = (file: File) => {
        setSelectedFile(file)
        onFileSelect(file)
    }

    const handleDrag = (e: React.DragEvent) => {
        e.preventDefault()
        e.stopPropagation()
        if (e.type === 'dragenter' || e.type === 'dragover') {
            setDragActive(true)
        } else if (e.type === 'dragleave') {
            setDragActive(false)
        }
    }

    const handleDrop = (e: React.DragEvent) => {
        e.preventDefault()
        e.stopPropagation()
        setDragActive(false)

        const file = e.dataTransfer.files?.[0]
        if (file && isCsv(file)) select(file)
    }

    const handleInput = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0]
        if (file) select(file)
    }

    return (
        <div
            className={cn(
                'border-2 border-dashed rounded-lg p-8 text-center transition-colors',
                dragActive ? 'border-primary bg-primary/5' : 'border-muted-foreground/25',
                disabled && 'opacity-75 pointer-events-none',
            )}
            onDragEnter={handleDrag}
            onDragLeave={handleDrag}
            onDragOver={handleDrag}
            onDrop={handleDrop}
        >
            <input
                type="file"
                accept=".csv"
                onChange={handleInput}
                className="hidden"
                id="csv-upload"
                disabled={disabled}
            />

            {selectedFile ? (
                <div className="flex flex-col items-center gap-2">
                    <FileText className="size-12 text-primary" />
                    <div className="flex items-center gap-2">
                        <span className="font-medium">{selectedFile.name}</span>
                        {!disabled && (
                            <Button
                                variant="ghost"
                                size="icon"
                                className="size-6"
                                aria-label={t('common:actions.clear')}
                                onClick={() => setSelectedFile(null)}
                            >
                                <X className="size-4" />
                            </Button>
                        )}
                    </div>
                    <span className="text-sm text-muted-foreground">
                        {(selectedFile.size / 1024 / 1024).toFixed(2)} MB
                    </span>
                </div>
            ) : (
                <label htmlFor="csv-upload" className="cursor-pointer">
                    <div className="flex flex-col items-center gap-2">
                        <Upload className="size-12 text-muted-foreground" />
                        <div className="text-lg font-medium">{t('import.dropzone')}</div>
                        <div className="text-sm text-muted-foreground">{t('import.dropzoneHint')}</div>
                    </div>
                </label>
            )}
        </div>
    )
}
