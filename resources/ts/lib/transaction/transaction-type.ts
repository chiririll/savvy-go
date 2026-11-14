/** Both kinds of transfer move money between accounts and have no category. */
export function isTransferType(type?: string | null): boolean {
    return type === 'transfer' || type === 'transfer_out'
}
