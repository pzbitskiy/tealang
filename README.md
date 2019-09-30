# Teal Language

ANTLR-based high level language for TEAL

## Prerequisites

1. Set up **ANTLR** as [explained](https://www.antlr.org/)
2. Set `CLASSPATH`
```
export CLASSPATH=.:$(pwd)/grammar:$CLASSPATH
```

## Build

```
antlr4 grammar/TealLang.g4 && javac grammar/TealLang*.java
```

## Run

```
cat examples/fee-reimburse.tl | grun TealLang prog -gui
```