import stdlib.const
import common

function getShardKey(shardNum) {
    return concat("s", shardNum)
}

function ensureShardByToken(tokenIn, targetShardStrIn) {
    let token = btoi(tokenIn)
    let targetShardIn = btoi(targetShardStrIn)
    let targetShard = token / tokensPerShard
    return targetShardIn == targetShard
}

function ensureShardAppByToken(tokenIn, targetShardStrIn) {
    let shardKey = getShardKey(targetShardStrIn)
    let targetShardAppId = apps[0].get(shardKey)
    // let targetShardAppId = app_global_get(shardKey)
    // ensure the second txn is shard app call
    if targetShardAppId == gtxn[1].ApplicationID && tokenIn == gtxn[1].ApplicationArgs[1] && txn.ApplicationArgs[0] == gtxn[1].ApplicationArgs[0] {
        // TODO: check txn.ForeignApps[0] == targetShardAppId
        return 1
    }
    return 0
}

function approval() {
    if txn.ApplicationID == 0 {
        // this must be a standalone transaction
        if global.GroupSize != 1 || txn.NumAppArgs != 1 || txn.ApplicationArgs[0] != "create" {
            return 0
        }
        apps[0].put(creatorKey, txn.Sender)
        apps[0].put(appIdKey, global.CurrentApplicationID)
        // app_global_put(creatorKey, txn.Sender)
        // app_global_put(appIdKey, global.CurrentApplicationID)
        return 1
    }

    if txn.OnCompletion != AcNoOp {
        return 0
    }

    // other commands require at least three args
    if txn.NumAppArgs < 3 {
        return 0
    }

    if txn.ApplicationArgs[0] == "shard_create" {
        let creator = apps[0].get(creatorKey)
        // let creator = app_global_get(creatorKey)
        if txn.Sender != creator || global.GroupSize != 1 {
            return 0
        }

        let createdShard = txn.ApplicationArgs[2]
        let shard = btoi(createdShard)
        let lastShard, exist = apps[0].getEx(lastShardKey)
        // let lastShard, exist = app_global_get_ex(0, lastShardKey)
        if !exist && shard != 0 || exist && lastShard != shard + 1 {
            return 0;
        }

        // save last shard
        apps[0].put(lastShardKey, shard)
        // app_global_put(lastShardKey, shard)

        // save sharding app idx
        let shardKey = getShardKey(createdShard)
        let shardAppId = btoi(txn.ApplicationArgs[1])
        apps[0].put(shardKey, shardAppId)
        // app_global_put(shardKey, shardAppId)
        return 1
    }

    if txn.ApplicationArgs[0] == "mint" {
        // must be first in a group
        if global.GroupSize != 2 || txn.GroupIndex != 0 {
            return 0
        }

        let creator = apps[0].get(creatorKey)
        // let creator = app_global_get(creatorKey)
        if txn.Sender != creator || gtxn[1].Sender != creator {
            return 0
        }

        let tokenIn = txn.ApplicationArgs[1]
        let targetShardStrIn = txn.ApplicationArgs[2]
        if !ensureShardByToken(tokenIn, targetShardStrIn) {
            return 0
        }

        if !ensureShardAppByToken(tokenIn, targetShardStrIn) {
            return 0
        }

        const foreignAppIdx = 0
        let value, exist = apps[0].getEx(tokenIn)
        // let value, exist = app_global_get_ex(foreignAppIdx, tokenIn)
        if exist {
            return 0
        }
        return 1
    }

    if txn.ApplicationArgs[0] == "transfer" {
        // must be first in a group
        if global.GroupSize != 2 || txn.GroupIndex != 0 {
            return 0
        }
        let tokenIn = txn.ApplicationArgs[1]
        let targetShardStrIn = txn.ApplicationArgs[2]
        let toIn = txn.ApplicationArgs[3]

        if !ensureShardByToken(tokenIn, targetShardStrIn) {
            return 0
        }
        if !ensureShardAppByToken(tokenIn, targetShardStrIn) {
            return 0
        }

        const foreignAppIdx = 0
        let value, exist = apps[0].getEx(tokenIn)
        // let value, exist = app_global_get_ex(foreignAppIdx, tokenIn)
        if !exist {
            return 0
        }
        let addr = substring(value, 32, 64)
        if addr == txn.Sender {
            return 1
        }
        if addr == global.ZeroAddress {
            let creator = apps[0].get(creatorKey)
            // let creator = app_global_get(creatorKey)
            if txn.Sender == creator {
                return 1
            }
        }
    }
    return 0
}
