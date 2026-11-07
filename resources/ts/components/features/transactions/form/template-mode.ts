import { createContext, useContext } from 'react'

/**
 * Whether the form draws the template of a recurring rule instead of a transaction: it has no date, and a transfer
 * may leave the amount received open for each occurrence to work out.
 */
export const TemplateModeContext = createContext(false)

export const useTemplateMode = () => useContext(TemplateModeContext)
