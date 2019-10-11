default: all

all:
	java -jar ./antlr-4.7.2-complete.jar -Dlanguage=Go -o gen/go Tealang.g4
	go build ./main.go && go test ./...

java-trace:
	java -jar ./antlr-4.7.2-complete.jar Tealang.g4 -o gen/java && javac gen/java/Tealang*.java
	java org.antlr.v4.gui.TestRig Tealang prog -diagnostics -trace examples/ex.tl -gui

java-gui:
	java -jar ./antlr-4.7.2-complete.jar Tealang.g4 -o gen/java && javac gen/java/Tealang*.java
	java org.antlr.v4.gui.TestRig Tealang prog -diagnostics examples/ex.tl -gui
