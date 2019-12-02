default: all

all: go grammar-java
test: go

ANTLR4_JAR := /usr/local/lib/antlr-4.7.2-complete.jar

grammar-all: grammar-go

grammar-go:
	java -jar $(ANTLR4_JAR) -Dlanguage=Go -o gen/go TealangLexer.l4
	java -jar $(ANTLR4_JAR) -Dlanguage=Go -o gen/go TealangParser.g4

grammar-java:
	java -jar $(ANTLR4_JAR) TealangLexer.l4 -o gen/java
	java -jar $(ANTLR4_JAR) TealangParser.g4 -o gen/java
	javac gen/java/Tealang*.java -classpath "gen/java:$(ANTLR4_JAR)"

go: grammar-go
	go generate ./compiler
	go build -o tealang ./main.go
	go test ./...

java-trace: grammar-java
	java -classpath "gen/java:$(ANTLR4_JAR)" org.antlr.v4.gui.TestRig Tealang program -diagnostics -trace examples/ex.tl

java-gui: grammar-java
	java -classpath "gen/java:$(ANTLR4_JAR)" org.antlr.v4.gui.TestRig Tealang program -diagnostics examples/w.tl -gui
