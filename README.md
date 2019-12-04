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

```sh
tealang -c mycontract.tl -o mycontract.teal
```

Checkout [syntax highlighter](https://github.com/pzbitskiy/tealang-syntax-highlighter) for VSCode.

TODO: Tealang guide

## Build from sources

### Prerequisites

1. Set up **ANTLR4** as explained in [the documentation](https://www.antlr.org/)
2. Install runtime for Go
    ```
    go get github.com/antlr/antlr4/runtime/Go/antlr
    ```

### Build
```sh
make go
```

### Build and run Java AST visualizer
```sh
make java-gui
```
