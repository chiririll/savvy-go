import { z } from 'zod'
import { descriptionSchema } from './transactions'

/** What the restricted form of a transaction edits: its title, tags, and the category and note of each line. */
export const transactionDetailsSchema = z.object({
    description: descriptionSchema,
    tag_ids: z.array(z.number()),
    lines: z.array(
        z.object({
            category_id: z.coerce.number().positive().nullable(),
            /** Markdown. */
            note: z.string().nullable(),
        }),
    ),
})

export type TransactionDetailsValues = z.infer<typeof transactionDetailsSchema>
