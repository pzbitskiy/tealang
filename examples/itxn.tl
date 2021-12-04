
const Pay = 1
const AssetConfig = 3

let _ = 1

function pay() {
    itxn.begin()
    itxn.TypeEnum = Pay
    itxn.Amount = 5000
    itxn.Receiver = txn.Sender
    itxn.submit()  
    return 1
}

function createAsset() {
    itxn.begin()
    itxn.TypeEnum = AssetConfig
    itxn.ConfigAssetTotal = 1000000
    itxn.ConfigAssetDecimals = 3
    itxn.ConfigAssetUnitName = "oz"
    itxn.ConfigAssetName = "Gold"
    itxn.ConfigAssetURL = "https://gold.rush/"
    itxn.ConfigAssetManager = global.CurrentApplicationAddress
    itxn.submit()
    apps[0].put("assetid", itxn.CreatedAssetID)
    return 1
}


function logic() {
  if txn.ApplicationArgs[0] == "payme" {
    _= pay()
  }
  if txn.ApplicationArgs[0] == "makeone" {
    _= createAsset()    
  }
  
  return 1
}
