default: all

all: go java-trace

ANTLR4_JAR := /usr/local/lib/antlr-4.7.2-complete.jar

go:
	java -jar $(ANTLR4_JAR) -Dlanguage=Go -o gen/go Tealang.g4
	go generate ./compiler
	go build -o tealang ./main.go
	go test ./...

java-trace:
	java -jar $(ANTLR4_JAR) Tealang.g4 -o gen/java
	javac gen/java/Tealang*.java -classpath "gen/java:$(ANTLR4_JAR)"
	java -classpath "gen/java:$(ANTLR4_JAR)" org.antlr.v4.gui.TestRig Tealang program -diagnostics -trace examples/ex.tl

java-gui:
	java -jar $(ANTLR4_JAR) Tealang.g4 -o gen/java
	javac gen/java/Tealang*.java -classpath "gen/java:$(ANTLR4_JAR)"
	java -classpath "gen/java:$(ANTLR4_JAR)" org.antlr.v4.gui.TestRig Tealang program -diagnostics examples/w.tl -gui
