import stdlib.const

// Suppose the owner of account A wants to send a payment to account 'to', but does not want to pay a transaction fee.
// If account A signs the following contract with the appropriate parameters,
// then anyone can cover a fee for that payment on account A's behalf.
// The contract works by approving a group of two transactions (meaning the two transactions will occur together or not at all).
// The first transaction must spend the transaction fee into account A,
// and the second transaction must be the specified payment transaction from account A to account 'to'.
function DynamicFee(to, amt, closeTo, firstValid, lastValid, lease) {
    const expectedGroupSize = 2
    const reimburseTxIndex = 0
    const myTxIndex = 1

    // ensure group is layout properly
    if global.GroupSize != expectedGroupSize || txn.GroupIndex != myTxIndex {
        error
    }

    // Check that the first transaction (reimbursement) is a payment,
    // which is required since the first transaction should be paying the fee for the second
    if gtxn[reimburseTxIndex].TypeEnum != TxTypePayment {
        return 0
    }

    // Check that the second transaction is a payment as well
    if txn.TypeEnum != TxTypePayment {
        return 0
    }

    // specify that the receiver of funds from the first transaction (reimbursement)
    // is equal to the sender of the second transaction
    // and funds are equal to fee of the second transaction
    let reimburseReceiver = gtxn[reimburseTxIndex].Receiver
    let reimburseAmount = gtxn[reimburseTxIndex].Amount
    let sender = txn.Sender
    let fee = txn.Fee
    if sender != reimburseReceiver || fee != reimburseAmount  {
        return 0
    }

    // verify that all other tx fields match to the contract parameters
    if txn.Receiver == to && txn.CloseRemainderTo == closeTo && txn.Amount == amt && txn.FirstValid == firstValid && txn.LastValid == lastValid && txn.Lease == lease {
        return 1
    }

    return 0
}