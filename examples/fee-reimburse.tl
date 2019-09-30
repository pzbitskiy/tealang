if gsize != 2 {
    error
}

let fee = txn.fee
let sender = txn.sender

// as a default assume the reimbursement transaction has index 0 in the group
let feeReceiver = gtx[0].receiver
let reimburseAmount = gtx[0].amount

// if current index is zero, then assume the reimbursement transaction has index 1
if gindex == 0 {
    feeReceiver = gtx[1].receiver
    reimburseAmount = gtx[1].amount
}

if sender != feeReceiver || fee != reimburseAmount {
    error
}

return 1
