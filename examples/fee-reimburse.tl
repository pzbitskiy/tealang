if global.GroupSize != 2 {
    error
}

let myFee = txn.Fee
let mySender = txn.Sender
let reimburseTxIndex = if txn.GroupIndex == 0 { 1 } else { 0 }

let reimburseTo = ""
let reimburseAmount = 0

// depending on reimbursement transaction index (0 or 1) determine values to check
if reimburseTxIndex == 1 {
    reimburseTo = gtxn[1].Receiver
    reimburseAmount = gtxn[1].Amount
} else {
    reimburseTo = gtxn[0].Receiver
    reimburseAmount = gtxn[0].Amount
}

if mySender != reimburseTo || myFee != reimburseAmount {
    return 0
}

return 1
