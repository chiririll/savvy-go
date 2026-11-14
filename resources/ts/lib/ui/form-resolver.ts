import { zodResolver } from '@hookform/resolvers/zod'
import type { FieldValues, Resolver } from 'react-hook-form'

export function schemaResolver<T extends FieldValues>(
    schema: Parameters<typeof zodResolver>[0],
): Resolver<T> {
    return zodResolver(schema) as Resolver<T>
}
