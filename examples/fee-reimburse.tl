if global.GroupSize != 2 {
    error
}

let fee = txn.Fee
let sender = txn.Sender

// as a default assume the reimbursement transaction has index 0 in the group
let feeReceiver = gtxn[0].Receiver
let reimburseAmount = gtxn[0].Amount

// if current index is zero, then assume the reimbursement transaction has index 1
if txn.GroupIndex == 0 {
    feeReceiver = gtxn[1].Receiver
    reimburseAmount = gtxn[1].Amount
}

if sender != feeReceiver || fee != reimburseAmount {
    return 0
}

return 1
