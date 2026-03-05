import { z } from 'zod'

/**
 * Zod schema for account selector value (AccountSelector uses { [id]: true } or null).
 * Use in entry dialog forms when an account is required.
 */
export const accountValidation = z
    .union([z.null(), z.record(z.boolean())])
    .refine((obj) => obj != null, { message: 'Account must be selected' })
