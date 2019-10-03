default: all

all:
	java -jar ./antlr-4.7.2-complete.jar -Dlanguage=Go -o gen/go Tealang.g4
	go run parser/main.go

java:
	java -jar ./antlr-4.7.2-complete.jar Tealang.g4 -o gen/java && javac gen/java/Tealang*.java
	java org.antlr.v4.gui.TestRig Tealang prog -diagnostics -trace examples/ex.tl
