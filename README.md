# Teal Language

High-level language for Algorand Smart Contracts at Layer-1 and it's low-level TEAL language.
The goal is to abstract the stack-based TEAL VM and provide imperative Go/JS/Python-like syntax.

## Language Features

* Integer and bytes types

* Variables and constants
```
let variable1 = 1
const myaddr = "XYZ"
```

* All binary and unary operations from TEAL
```
let a = (1 + 2) / 3
let b = ~a
```

* Inlined functions
```
function sample(a) {
    return a - 1
}

function logic() {
    return sample(2)
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
    a = test()
    return a
}
```

* Antlr-based parser

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
* Stdin to Stdout
    ```sh
    cat mycontract.tl | tealang -s -r - > mycontract.tok
    ```

Checkout [syntax highlighter](https://github.com/pzbitskiy/tealang-syntax-highlighter) for vscode.

TODO: Tealang guide

## Build from sources

### Prerequisites

1. Set up **ANTLR4** as explained in [the documentation](https://www.antlr.org/)
2. Install runtime for Go
    ```sh
    go get github.com/antlr/antlr4/runtime/Go/antlr
    ```
3. Install and setup **go-algorand**. Read [Algorand README](https://github.com/algorand/go-algorand/blob/master/README.md) if needed.
    ```sh
    go get github.com/algorand/go-algorand
    pushd $(go env GOPATH)/src/github.com/algorand/go-algorand
    make
    cd $(go env GOPATH)/src/github.com/satori/go.uuid
    git checkout v1.2.0
    popd
    ```

### Build
```sh
make go
```

### Build and run Java AST visualizer
```sh
make java-gui
```
