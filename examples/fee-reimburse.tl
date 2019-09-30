if gsize != 2 {
    error
}

let fee = txn.fee
let sender = txn.sender

let feeReceiver = gtx[0].receiver
let reimburseAmount = gtx[0].amount
if gindex != 0 {
    feeReceiver = gtx[1].receiver
    reimburseAmount = gtx[1].amount
}

if sender == feeReceiver && fee == reimburseAmount {
    return 1
}
return 0
