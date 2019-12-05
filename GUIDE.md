# Tealang guide

Tealang translates all (except `mulw`) own constructions to corresponding **TEAL** instructions
so that there is almost one-to-one mapping between statements in both languages.

Refer to [TEAL documentation](https://developer.algorand.org/docs/teal) for details.

*Note, all code snippets below only for Tealang features demonstration.*

## Program structure

* imports
* variable and constant declarations and function definitions
* logic function

## Types

* uint64 (unsigned integer)
* []byte (byte array)

## Statements vs expressions

Statement is a standalone unit of execution that does not return any value.
In opposite, expression is evaluated to some value.

## Declarations, definitions and assignments

Constants, variables and function are supported:
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

Unlike **TEAL** functions are supported and inlined at the calling point.
Functions must return some value and can not be recursive.

```
function inc(x) { return x+1; }
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

## Builtin objects

There are 4 builtin objects: `txn`, `gtxn`, `global`, `args`. Accessing them is an expression.

| Object and Syntax | Description |
| --- | --- |
| `args[N]` | returns Args[N] value |
| `txn.FIELD` | retrieves field from current transaction |
| `gtxn[N].FIELD` | retrieves field from a transaction N in the current transaction group |
| `global.FIELD` | returns globals |

#### Transaction fields
| Index | Name | Type | Notes |
| --- | --- | --- | --- |
| 0 | Sender | []byte | 32 byte address |
| 1 | Fee | uint64 | micro-Algos |
| 2 | FirstValid | uint64 | round number |
| 3 | FirstValidTime | uint64 | Causes program to fail; reserved for future use. |
| 4 | LastValid | uint64 | round number |
| 5 | Note | []byte |  |
| 6 | Lease | []byte |  |
| 7 | Receiver | []byte | 32 byte address |
| 8 | Amount | uint64 | micro-Algos |
| 9 | CloseRemainderTo | []byte | 32 byte address |
| 10 | VotePK | []byte | 32 byte address |
| 11 | SelectionPK | []byte | 32 byte address |
| 12 | VoteFirst | uint64 |  |
| 13 | VoteLast | uint64 |  |
| 14 | VoteKeyDilution | uint64 |  |
| 15 | Type | []byte |  |
| 16 | TypeEnum | uint64 | See table below |
| 17 | XferAsset | uint64 | Asset ID |
| 18 | AssetAmount | uint64 | value in Asset's units |
| 19 | AssetSender | []byte | 32 byte address. Causes clawback of all value of asset from AssetSender if Sender is the Clawback address of the asset. |
| 20 | AssetReceiver | []byte | 32 byte address |
| 21 | AssetCloseTo | []byte | 32 byte address |
| 22 | GroupIndex | uint64 | Position of this transaction within an atomic transaction group. A stand-alone transaction is implicitly element 0 in a group of 1. |
| 23 | TxID | []byte | The computed ID for this transaction. 32 bytes. |

#### Global fields

| Index | Name | Type | Notes |
| --- | --- | --- | --- |
| 0 | MinTxnFee | uint64 | micro Algos |
| 1 | MinBalance | uint64 | micro Algos |
| 2 | MaxTxnLife | uint64 | rounds |
| 3 | ZeroAddress | []byte | 32 byte address of all zero bytes |
| 4 | GroupSize | uint64 | Number of transactions in this atomic transaction group. At least 1. |

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