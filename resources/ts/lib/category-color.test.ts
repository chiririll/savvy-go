import { describe, expect, it } from 'vitest'

import { categoryIconStyle } from './category-color'

describe('categoryIconStyle', () => {
    it('tints the background with the category colour', () => {
        expect(categoryIconStyle('#ff8800')).toEqual({ backgroundColor: '#ff880033' })
    })

    it('returns no style without a colour', () => {
        expect(categoryIconStyle()).toBeUndefined()
        expect(categoryIconStyle(null)).toBeUndefined()
        expect(categoryIconStyle('')).toBeUndefined()
    })
})
