import type { TFunction } from 'i18next'
import type { PayloadError } from '@/lib/transaction-payload'
import type { TransferError } from '@/lib/transfer-payload'

/** What to tell the user about a payload the API would refuse, by the part of the form it belongs to. */
export function payloadErrorMessage(t: TFunction, error: PayloadError): string {
    switch (error) {
        case 'unknownAccount':
            return t('validation.selectAccount')
        case 'duplicateAccount':
            return t('validation.accountUsedTwice')
        case 'amountZero':
            return t('validation.amountPositive')
        case 'netSign':
            return t('validation.changeTooLarge')
        case 'lineCurrency':
            return t('validation.lineCurrency')
    }
}

export function transferErrorMessage(t: TFunction, error: TransferError): string {
    switch (error) {
        case 'unknownSpace':
            return t('validation.selectSpace')
        case 'unknownFrom':
            return t('validation.selectAccount')
        case 'unknownTo':
            return t('validation.transferDestination')
        case 'sameAccount':
            return t('validation.accountUsedTwice')
        case 'amountZero':
        case 'receivedZero':
        case 'feeInvalid':
            return t('validation.amountPositive')
    }
}
