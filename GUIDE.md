# Tealang guide

Tealang translates all own constructions to corresponding **TEAL** instructions
so that there is almost one-to-one mapping between statements in both languages.

Refer to [TEAL documentation](https://developer.algorand.org/docs/reference/teal/specification) for details.

*Note, all code snippets below only for Tealang features demonstration.*

## Program structure

* imports
* variable and constant declarations and function definitions
* logic function

## Types

* uint64 (unsigned integer)
* []byte (byte array)

In some circumstances you can use `toint()` or `tobyte()` to specify an unknown type:


```
let a = accounts[0].get("key") // type unknown
let b = toint(a) + 1

```

## Statements vs expressions

Statement is a standalone unit of execution that does not return any value.
In opposite, expression is evaluated to some value.

## Declarations, definitions and assignments

Constants, variables and functions are supported:
```
const a = 1
const b = "abc\x01"
let x = b
function test(x) { return x; }
```

Declarations, definitions and assignments are statements.

## String literals

String literals are decoded and stored as byte arrays in underlying **TEAL** program.
Literals might be encoded and contain hex escape sequence. The following encoding prefixes are supported:
* b32 for **base32** strings
* b64 for **base64** strings
* addr for Algorand addresses

```
const zeroAddress = addr"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY5HFKQ"
const someval = b64"MTIz"

function logic() {
    if txn.Receiver == zeroAddress {
        return 1
    }
    return 0
}
```

## Functions

Functions must return some value and can not be recursive or re-entrant.

```
inline function inc(x) { return x+1; }  // inlined at the calling point
function dec(y) { return y-1; }         // uses callsub and retsub
function logic() { return inc(0); } 
```

## Logic function

Must exist in every program and return integer. The return value (zero/non-zero) is **TRUE** or **FALSE** return code for entire **TEAL** program (smart contract).
```
function logic() {
    if txn.Sender == addr"47YPQTIGQEO7T4Y4RWDYWEKV6RTR2UNBQXBABEEGM72ESWDQNCQ52OPASU" {
         return 1
    }
    return 0
}
```

## Flow control statements

### if-else

Consist of `if` keyword, conditional expression (must evaluate to integer), `if-block` and optional `else-block`.
```
if x == 1 {
    return 1
} else {
    let x = txn.Receiver
}
```

### return

`return` forces current function to exit and return a value. For the special `logic` function it would be entire program return value.

### error

`error` forces program to exit with an error.

## Comments

Single line comments are supported, commenting sequence is `//`

## Expressions

### Conditional expression

Similar to conditional statement but `else-block` is required, and both blocks must evaluate to an expression.
```
let sender = if global.GroupSize > 1 { txn.Sender } else { gtxn[1].Sender }
```

### Arithmetic, Logic, and Cryptographic Operations

All operations like +, -, *, ==, !=, <, >, >=, etc.
See [TEAL documentation](https://github.com/algorand/go-algorand/blob/master/data/transactions/logic/README.md#arithmetic-logic-and-cryptographic-operations) for the full list.

## Builtin functions

`sha256`, `keccak256`, `sha512_256`, `ed25519verify`, `len`, `itob`, `btoi`, `mulw`, `addw`, `concat`, `substring3`, `assert`, `expw`, `exp`, `getbit`, `getbyte`, `setbit`, `setbyte`, `shl`, `shr`, `bitlen`, `sqrt`, `log` are supported.
`mulw`, `addw`, `expw` are special - they return two values, high and low.

```
let h = len("123")
let l = btoi("1")
h, l = mulw(l, h)
```

## Builtin objects

There are 8 builtin objects: `txn`, `gtxn`, `itxn`, `global`, `args`, `assets`, `accounts`, `apps`.

| Object and Syntax | Description |
| --- | --- |
| `args[N]` | returns LogicSig Args[N] value as []byte |
| `txn.FIELD` | retrieves field from current transaction (see below) |
| `gtxn[N].FIELD` | retrieves field from a transaction N in the current transaction group |
| `itxn.begin()/submit()\|FIELD` | create/submit inner transaction or set field (see below) |
| `global.FIELD` | returns globals (see below) |
| `assets[N].FIELD` | returns asset information for an asset specified by `txn.ForeignAssets[N]` (see below) |
| `accounts[N].Balance\|MinBalance` | returns balance (min balance) of an account specified by `txn.Accounts[N-1]`, N=0 for txn.Sender |
| `accounts[N].method` | returns state data of an account specified by `txn.Accounts[N-1]`, N=0 for txn.Sender (see below) |
| `apps[N].method` | returns application global state data for an app specified by `txn.ForeignApps[N-1]`, N=0 means this app (see below) |

#### Transaction fields

| Index | Name | Type | Notes |
| --- | --- | --- | --- |
| 0 | Sender | []byte | 32 byte address |
| 1 | Fee | uint64 | micro-Algos |
| 2 | FirstValid | uint64 | round number |
| 3 | FirstValidTime | uint64 | Causes program to fail; reserved for future use. |
| 4 | LastValid | uint64 | round number |
| 5 | Note | []byte | Note field |
| 6 | Lease | []byte | Lease field |
| 7 | Receiver | []byte | 32 byte address |
| 8 | Amount | uint64 | micro-Algos |
| 9 | CloseRemainderTo | []byte | 32 byte address |
| 10 | VotePK | []byte | 32 byte address |
| 11 | SelectionPK | []byte | 32 byte address |
| 12 | VoteFirst | uint64 |  |
| 13 | VoteLast | uint64 |  |
| 14 | VoteKeyDilution | uint64 |  |
| 15 | Type | []byte | transaction type string |
| 16 | TypeEnum | uint64 | type constant |
| 17 | XferAsset | uint64 | Asset ID |
| 18 | AssetAmount | uint64 | value in Asset's units |
| 19 | AssetSender | []byte | 32 byte address. Causes clawback of all value of asset from AssetSender if Sender is the Clawback address of the asset. |
| 20 | AssetReceiver | []byte | 32 byte address |
| 21 | AssetCloseTo | []byte | 32 byte address |
| 22 | GroupIndex | uint64 | Position of this transaction within an atomic transaction group. A stand-alone transaction is implicitly element 0 in a group of 1. |
| 23 | TxID | []byte | The computed ID for this transaction. 32 bytes. |
| 24 | ApplicationID | uint64 | ApplicationID from ApplicationCall transaction. LogicSigVersion >= 2. |
| 25 | OnCompletion | uint64 | ApplicationCall transaction on completion action. LogicSigVersion >= 2. |
| 26 | ApplicationArgs | []byte | Arguments passed to the application in the ApplicationCall transaction. LogicSigVersion >= 2. |
| 27 | NumAppArgs | uint64 | Number of ApplicationArgs. LogicSigVersion >= 2. |
| 28 | Accounts | []byte | Accounts listed in the ApplicationCall transaction. LogicSigVersion >= 2. |
| 29 | NumAccounts | uint64 | Number of Accounts. LogicSigVersion >= 2. |
| 30 | ApprovalProgram | []byte | Approval program. LogicSigVersion >= 2. |
| 31 | ClearStateProgram | []byte | Clear state program. LogicSigVersion >= 2. |
| 32 | RekeyTo | []byte | 32 byte Sender's new AuthAddr. LogicSigVersion >= 2. |
| 33 | ConfigAsset | uint64 | Asset ID in asset config transaction. LogicSigVersion >= 2. |
| 34 | ConfigAssetTotal | uint64 | Total number of units of this asset created. LogicSigVersion >= 2. |
| 35 | ConfigAssetDecimals | uint64 | Number of digits to display after the decimal place when displaying the asset. LogicSigVersion >= 2. |
| 36 | ConfigAssetDefaultFrozen | uint64 | Whether the asset's slots are frozen by default or not, 0 or 1. LogicSigVersion >= 2. |
| 37 | ConfigAssetUnitName | []byte | Unit name of the asset. LogicSigVersion >= 2. |
| 38 | ConfigAssetName | []byte | The asset name. LogicSigVersion >= 2. |
| 39 | ConfigAssetURL | []byte | URL. LogicSigVersion >= 2. |
| 40 | ConfigAssetMetadataHash | []byte | 32 byte commitment to some unspecified asset metadata. LogicSigVersion >= 2. |
| 41 | ConfigAssetManager | []byte | 32 byte address. LogicSigVersion >= 2. |
| 42 | ConfigAssetReserve | []byte | 32 byte address. LogicSigVersion >= 2. |
| 43 | ConfigAssetFreeze | []byte | 32 byte address. LogicSigVersion >= 2. |
| 44 | ConfigAssetClawback | []byte | 32 byte address. LogicSigVersion >= 2. |
| 45 | FreezeAsset | uint64 | Asset ID being frozen or un-frozen. LogicSigVersion >= 2. |
| 46 | FreezeAssetAccount | []byte | 32 byte address of the account whose asset slot is being frozen or un-frozen. LogicSigVersion >= 2. |
| 47 | FreezeAssetFrozen | uint64 | The new frozen value, 0 or 1. LogicSigVersion >= 2. |
| 48 | Assets | uint64 | Foreign Assets listed in the ApplicationCall transaction. LogicSigVersion >= 3. |
| 49 | NumAssets | uint64 | Number of Assets. LogicSigVersion >= 3. |
| 50 | Applications | uint64 | Foreign Apps listed in the ApplicationCall transaction. LogicSigVersion >= 3. |
| 51 | NumApplications | uint64 | Number of Applications. LogicSigVersion >= 3. |
| 52 | GlobalNumUint | uint64 | Number of global state integers in ApplicationCall. LogicSigVersion >= 3. |
| 53 | GlobalNumByteSlice | uint64 | Number of global state byteslices in ApplicationCall. LogicSigVersion >= 3. |
| 54 | LocalNumUint | uint64 | Number of local state integers in ApplicationCall. LogicSigVersion >= 3. |
| 55 | LocalNumByteSlice | uint64 | Number of local state byteslices in ApplicationCall. LogicSigVersion >= 3. |

#### Global fields

| Index | Name | Type | Notes |
| --- | --- | --- | --- |
| 0 | MinTxnFee | uint64 | micro Algos |
| 1 | MinBalance | uint64 | micro Algos |
| 2 | MaxTxnLife | uint64 | rounds |
| 3 | ZeroAddress | []byte | 32 byte address of all zero bytes |
| 4 | GroupSize | uint64 | Number of transactions in this atomic transaction group. At least 1. |
| 5 | LogicSigVersion | uint64 | Maximum supported TEAL version. LogicSigVersion >= 2. |
| 6 | Round | uint64 | Current round number. LogicSigVersion >= 2. |
| 7 | LatestTimestamp | uint64 | Last confirmed block UNIX timestamp. Fails if negative. LogicSigVersion >= 2. |
| 8 | CurrentApplicationID | uint64 | ID of current application executing. Fails if no such application is executing. LogicSigVersion >= 2. |
| 9 | CreatorAddress | []byte | Address of the creator of the current application. Fails if no such application is executing. LogicSigVersion >= 3. |
| 10 | CurrentApplicationAddress | []byte | Address of the current application. Fails if no such application is executing. LogicSigVersion >= 5. |


#### Asset fields

| Index | Name | Type | Notes |
| --- | --- | --- | --- |
| 0 | AssetTotal | uint64 | Total number of units of this asset |
| 1 | AssetDecimals | uint64 | See AssetParams.Decimals |
| 2 | AssetDefaultFrozen | uint64 | Frozen by default or not |
| 3 | AssetUnitName | []byte | Asset unit name |
| 4 | AssetName | []byte | Asset name |
| 5 | AssetURL | []byte | URL with additional info about the asset |
| 6 | AssetMetadataHash | []byte | Arbitrary commitment |
| 7 | AssetManager | []byte | Manager commitment |
| 8 | AssetReserve | []byte | Reserve address |
| 9 | AssetFreeze | []byte | Freeze address |
| 10 | AssetClawback | []byte | Clawback address |

#### Accounts methods

| Signature | Param Types | Return Types | Notes |
| --- | --- | --- | --- |
| assetBalance(assetId) | uint64 | uint64 | Amount of the asset unit held by this account |
| assetIsFrozen(assetId) | uint64 | uint64 | Is the asset frozen or not |
| optedIn(appId) | uint64 | uint64 | Returns 1 if opted in the app, and 0 otherwise, see [`app_opted_in` opcode](https://developer.algorand.org/docs/reference/teal/specification/#state-access) for details |
| getEx(appIdx, key) | uint64, []byte | any, uint64 (top) | Returns value and isOk flag, see [`app_local_get_ex` opcode](https://developer.algorand.org/docs/reference/teal/specification/#state-access) for details |
| get(key) | []byte | any | Returns value or 0 if does not exist, see [`app_local_get` opcode](https://developer.algorand.org/docs/reference/teal/specification/#state-access) for details |
| put(key, value) | []byte, any | - | Stores key-value pair in app's local store, does not return. See [`app_local_put` opcode](https://developer.algorand.org/docs/reference/teal/specification/#state-access) for details |
| del(key) | []byte | - | Deletes from app's local store, does not return. See [`app_local_del` opcode](https://developer.algorand.org/docs/reference/teal/specification/#state-access) for details |

#### Apps methods

| Signature | Param Types | Return Types | Notes |
| --- | --- | --- | --- |
| getEx(appIdx, key) | uint64, []byte | any, uint64 (top) | Returns value and isOk flag, see [`app_global_get_ex` opcode](https://developer.algorand.org/docs/reference/teal/specification/#state-access) for details |
| get(key) | []byte | any | Returns value or 0 if does not exist, see [`app_global_get` opcode](https://developer.algorand.org/docs/reference/teal/specification/#state-access) for details |
| put(key, value) | []byte, any | - | Stores key-value pair in app's global store, does not return. See [`app_global_put` opcode](https://developer.algorand.org/docs/reference/teal/specification/#state-access) for details |
| del(key) | []byte | - | Deletes from app's local store, does not return. See [`app_global_del` opcode](https://developer.algorand.org/docs/reference/teal/specification/#state-access) for details |

#### Inner Transactions

Asset creation example:

```
const AssetConfig = 3

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
```

Payment example:

```
const Pay = 1

itxn.begin()
itxn.TypeEnum = Pay
itxn.Amount = 5000
itxn.Receiver = txn.Sender
itxn.submit()  
```

| Index | Name | Type | Notes |
| --- | --- | --- | --- |
| 0 | Sender | []byte | 32 byte address |
| 1 | Fee | uint64 | micro-Algos |
| 2 | Receiver | []byte | 32 byte address |
| 3 | Amount | uint64 | micro-Algos |
| 4 | CloseRemainderTo | []byte | 32 byte address |
| 5 | Type | []byte | transaction type string |
| 6 | TypeEnum | uint64 | type constant |
| 7 | XferAsset | uint64 | Asset ID |
| 8 | AssetAmount | uint64 | value in Asset's units |
| 9 | AssetSender | []byte | 32 byte address. Causes clawback of all value of asset from AssetSender if Sender is the Clawback address of the asset. |
| 10 | AssetReceiver | []byte | 32 byte address |
| 11 | AssetCloseTo | []byte | 32 byte address |
| 12 | ConfigAsset | uint64 | Asset ID in asset config transaction. LogicSigVersion >= 2. |
| 13 | ConfigAssetTotal | uint64 | Total number of units of this asset created. LogicSigVersion >= 2. |
| 14 | ConfigAssetDecimals | uint64 | Number of digits to display after the decimal place when displaying the asset. LogicSigVersion >= 2. |
| 15 | ConfigAssetUnitName | []byte | Unit name of the asset. LogicSigVersion >= 2. |
| 16 | ConfigAssetName | []byte | The asset name. LogicSigVersion >= 2. |
| 17 | ConfigAssetURL | []byte | URL. LogicSigVersion >= 2. |
| 18 | ConfigAssetManager | []byte | 32 byte address. LogicSigVersion >= 2. |
| 19 | ConfigAssetReserve | []byte | 32 byte address. LogicSigVersion >= 2. |
| 20 | ConfigAssetFreeze | []byte | 32 byte address. LogicSigVersion >= 2. |
| 21 | ConfigAssetClawback | []byte | 32 byte address. LogicSigVersion >= 2. |
| 22 | FreezeAsset | uint64 | Asset ID being frozen or un-frozen. LogicSigVersion >= 2. |
| 23 | FreezeAssetAccount | []byte | 32 byte address of the account whose asset slot is being frozen or un-frozen. LogicSigVersion >= 2. |
| 24 | FreezeAssetFrozen | uint64 | The new frozen value, 0 or 1. LogicSigVersion >= 2. |

## Scopes

Tealang maintains a single global scope (shared between main program and imported modules) and nested scopes for every execution block.
Blocks are created for functions and if-else branches.
Parent scope is accessible from nested blocks. If a variable declared in nested block, it might shadow variable with the same name from parent scope.

```
let x = 1
function logic() {
    let x = 2       // shadows 1 in logic block
    if 1 {
        let x = 3   // shadows 2 in if-block
    }
    return x        // 2
}
```

## Imports

Unlike **TEAL**, a tealang program can be split to modules. There is a standard library `stdlib` containing some constants and **TEAL** templates as tealang functions.

### Module structure

* imports
* declarations and definitions

```
const myconst = 1
function myfunction() { return 0; }
```

## Standard library

At the moment consist of 2 files:
1. const.tl
2. template.tl

```
import stdlib.const

function logic() {
    let ret = TxTypePayment
    return ret
}
```

## More examples

* [examples directory](https://github.com/pzbitskiy/tealang/tree/master/examples)
* [stdlib directory](https://github.com/pzbitskiy/tealang/tree/master/stdlib)
