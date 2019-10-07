if global.GroupSize != 2 {
    error
}

let fee = txn.Fee
let sender = txn.Sender

// as a default assume the reimbursement transaction has index 0 in the group
let feeReceiver = gtx[0].Receiver
let reimburseAmount = gtx[0].Amount

// if current index is zero, then assume the reimbursement transaction has index 1
if txn.GroupIndex == 0 {
    feeReceiver = gtx[1].Receiver
    reimburseAmount = gtx[1].Amount
}

if sender != feeReceiver || fee != reimburseAmount {
    error
}

return 1
