# Teal Language

ANTLR-based high level language for TEAL

## Build

```
antlr4 grammar/TealLang.g4 && javac grammar/TealLang*.java
```

## Run

```
cd grammar
cat ../examples/fee-reimburse.tl | grun TealLang prog -gui
```