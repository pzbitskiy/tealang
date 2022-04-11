import stdlib.const

const seller_key = "seller"
const nft_id_key = "nft_id"
const start_time_key = "start"
const end_time_key = "end"
const reserve_amount_key = "reserve_amount"
const min_bid_increment_key = "min_bid_inc"
const num_bids_key = "num_bids"
const lead_bid_amount_key = "bid_amount"
const lead_bid_account_key = "bid_account"

function closeNFTTo(asset_id, acc) {
    let cur_addr = global.CurrentApplicationAddress
    let _, ok = accounts[cur_addr].assetBalance(asset_id)

    if ok {
        itxn.begin()
            itxn.TypeEnum = TxTypeAssetTransfer
            itxn.XferAsset = asset_id
            itxn.AssetCloseTo = acc
        itxn.submit()
    }

    return 1
}

function repayPreviousLeadBidder(prevLeadBidder, prevLeadBidAmount) {
    itxn.begin()
        itxn.TypeEnum = TxTypePayment
        itxn.Amount = prevLeadBidAmount - global.MinTxnFee
        itxn.Receiver = prevLeadBidder
    itxn.submit()

    return 1
}

function closeAccountTo(acc) {
    let cur_addr = global.CurrentApplicationAddress
    if accounts[cur_addr].Balance != 0 {
        itxn.begin()
            itxn.TypeEnum = TxTypePayment
            itxn.CloseRemainderTo = acc
        itxn.submit()
    }

    return 1
}

function onCreate(on_create_start_time, on_create_end_time) {
    apps[0].put(seller_key, txn.ApplicationArgs[0])
    apps[0].put(nft_id_key, btoi(txn.ApplicationArgs[1]))
    apps[0].put(start_time_key, on_create_start_time)
    apps[0].put(end_time_key, on_create_end_time)
    apps[0].put(reserve_amount_key, btoi(txn.ApplicationArgs[4]))
    apps[0].put(min_bid_increment_key, btoi(txn.ApplicationArgs[5]))
    apps[0].put(lead_bid_account_key, global.ZeroAddress)

    assert(global.LatestTimestamp < on_create_start_time)
    assert(on_create_start_time < on_create_end_time)

    return 1
}

function onSetup() {
    assert(global.LatestTimestamp < toint(apps[0].get(start_time_key)))
    itxn.begin()
        itxn.TypeEnum = TxTypeAssetTransfer
        itxn.XferAsset = toint(apps[0].get(nft_id_key))
        itxn.AssetReceiver = global.CurrentApplicationAddress
    itxn.submit()

    return 1
}

function onBid(on_bid_txn_index, on_bid_nft_holding) {
    assert(on_bid_nft_holding > 0)
    assert(toint(apps[0].get(start_time_key)) <= global.LatestTimestamp)
    assert(global.LatestTimestamp <= toint(apps[0].get(end_time_key)))
    assert(gtxn[on_bid_txn_index].TypeEnum == TxTypePayment)
    assert(gtxn[on_bid_txn_index].Sender == txn.Sender)
    assert(gtxn[on_bid_txn_index].Receiver == global.CurrentApplicationAddress)
    assert(gtxn[on_bid_txn_index].Amount == global.MinTxnFee)

    let min_amount = toint(apps[0].get(lead_bid_amount_key)) + toint(apps[0].get(min_bid_increment_key))
    if gtxn[on_bid_txn_index].Amount >= min_amount {
        if apps[0].get(lead_bid_account_key) != global.ZeroAddress {
            let acc = tobyte(apps[0].get(lead_bid_account_key))
            let amount = toint(apps[0].get(lead_bid_amount_key))
            let ok = repayPreviousLeadBidder(acc, amount)
        }

        apps[0].put(lead_bid_amount_key, gtxn[on_bid_txn_index].Amount)
        apps[0].put(lead_bid_account_key, gtxn[on_bid_txn_index].Sender)
        apps[0].put(num_bids_key, toint(apps[0].get(num_bids_key)) + 1)
        return 1
    }

    return 0
}

function onDelete() {
    let nft_id_int = toint(apps[0].get(nft_id_key))
    let lead_bid_account_byte = tobyte(apps[0].get(seller_key))
    let seller_byte = tobyte(apps[0].get(seller_key))
    if global.LatestTimestamp < toint(apps[0].get(start_time_key)) {
        assert(txn.Sender == tobyte(apps[0].get(seller_key)) || txn.Sender == global.CreatorAddress)
        let ok = closeNFTTo(nft_id_int, lead_bid_account_byte)
        ok = closeAccountTo(tobyte(apps[0].get(seller_key)))
        return 1
    }

    if toint(apps[0].get(end_time_key)) <= global.LatestTimestamp {
        
        if apps[0].get(lead_bid_account_key) != global.ZeroAddress {
            if apps[0].get(lead_bid_amount_key) >= apps[0].get(reserve_amount_key) {
                let ok = closeNFTTo(nft_id_int, lead_bid_account_byte)
            } else {
                let ok = closeNFTTo(nft_id_int, seller_byte)
                let acc = tobyte(apps[0].get(lead_bid_account_key))
                ok = repayPreviousLeadBidder(acc, toint(apps[0].get(lead_bid_amount_key)))
            }
        } else {
            let ok = closeNFTTo(nft_id_int, seller_byte)
        }

        let ok = closeAccountTo(tobyte(apps[0].get(seller_key)))
        return 1
    }

    return 0
}

function approval() {
    if (txn.ApplicationID == 0) {
        let on_create_start_time = btoi(txn.ApplicationArgs[2])
        let on_create_end_time = btoi(txn.ApplicationArgs[3])
        return onCreate(on_create_start_time, on_create_end_time)
    }
    
    // noop
    if txn.OnCompletion == 0 {
        let on_call_method = txn.ApplicationArgs[0]
        if on_call_method == "setup" {
            return onSetup()
        }
        if on_call_method == "bid" {
            let addr = global.CurrentApplicationAddress
            let on_bid_nft_holding, _ = accounts[addr].assetBalance(apps[0].get(nft_id_key))
            return onBid(txn.GroupIndex - 1, on_bid_nft_holding)
        }
    }

    // delete
    if txn.OnCompletion == 5 {
        return onDelete()
    }

    return 0
}