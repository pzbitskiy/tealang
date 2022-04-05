# Tealang

High-level language for Algorand Smart Contracts at Layer-1 and its low-level **TEAL v4** language.
The goal is to abstract the stack-based **TEAL** VM and provide imperative Go/JS/Python-like syntax.

## Language Features

* Integer and bytes types

* Variables and constants
```
let variable1 = 1
const myaddr = addr"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY5HFKQ"
```

* All binary and unary operations from **TEAL**
```
let a = (1 + 2) / 3
let b = ~a
```

* Functions
```
inline function sample1(a) {
    return a - 1
}

inline function sample2(a) {
    return a + 1
}

function logic() {
    return sample1(2) + sample2(3)
}
```

* Condition statements and expressions
```
function condition(a) {
    let b = if a == 1 { 10 } else { 0 }

    if b == 0 {
        return a
    }
    return 1
}
```

* Type checking
```
function get_string() {
    return "\x32\x33\x34"
}

function logic() {
    let a = 1
    a = get_string()
    return a
}
```

* Modules
```
import stdlib.const
```

* Antlr-based parser

## Language guide

[Documentation](GUIDE.md)

## Usage

* Tealang to bytecode
    ```sh
    tealang mycontract.tl -o mycontract.tok
    ```

* Tealang to TEAL
    ```sh
    tealang -c mycontract.tl -o mycontract.teal
    ```
* Tealang logic one-liner to bytecode
    ```sh
    tealang -l '(txn.Sender == "abc") && global.MinTxnFee > 2000' -o mycontract.tok
    ```
* stdin to stdout
    ```sh
    cat mycontract.tl | tealang -s -r - > mycontract.tok
    ```
* Dryrun / trace
    ```sh
    tealang -s -c -d '' examples/basic.tl
    ```

Checkout [syntax highlighter](https://github.com/pzbitskiy/tealang-syntax-highlighter) for vscode.

## Build from sources

### Prerequisites

1. Set up **ANTLR4** as explained in [the documentation](https://www.antlr.org/)
   Or run `make antlr-install`
2. Install runtime for Go
    ```sh
    go get -u github.com/antlr/antlr4/runtime/Go/antlr
    ```
3. Install and setup **go-algorand**. Read [Algorand README](https://github.com/algorand/go-algorand/blob/master/README.md) if needed.
    ```sh
    make algorand-install
    ```
### Build and test
```sh
make
```

### Build and run Java AST visualizer
```sh
make java-gui ARGS=examples/basic.tl
```

## Roadmap

1. Constant folding.
2. Improve errors reporting.
3. Code gen: do not use temp scratch in "assign and use" case.
4. Code gen: keep track scratch slots and mark as available after freeing with `load`.
