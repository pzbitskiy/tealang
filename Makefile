default: build

all: go test grammar-java
build: go

algorand-link:
	SYM_DIR_LOC=$$(go env GOPATH)/pkg/mod/$$(echo $$(grep -F 'github.com/algorand/go-algorand' go.mod | sed 's/ /@/' | cut -d " " -f1 )) ; \
	SYM_DIR_TARG=$$(go env GOPATH)/src/github.com/algorand/go-algorand; \
	[ ! -d "$$SYM_DIR_TARG" ] && echo "error: target directory not found ($$SYM_DIR_TARG)" && exit 1; \
	mkdir -p "$$(dirname "$$SYM_DIR_LOC")"; \
	[ -e "$$SYM_DIR_LOC" ] && mv "$$SYM_DIR_LOC" "$${SYM_DIR_LOC}__BACKUP__"; \
	echo "targ=$$SYM_DIR_TARG" "linkloc=$$SYM_DIR_LOC"; \
	ln -s "$$SYM_DIR_TARG" "$$SYM_DIR_LOC"

ANTLR4_VER := 4.9.3
ANTLR4_JAR := /usr/local/lib/antlr-$(ANTLR4_VER)-complete.jar

setup-antlr:
	sudo curl -o $(ANTLR4_JAR) https://www.antlr.org/download/antlr-$(ANTLR4_VER)-complete.jar
	export CLASSPATH="$(ANTLR4_JAR):$$CLASSPATH"

grammar-all: grammar-go

grammar-go:
	java -jar $(ANTLR4_JAR) -Dlanguage=Go -o gen/go TealangLexer.l4
	java -jar $(ANTLR4_JAR) -Dlanguage=Go -o gen/go TealangParser.g4

grammar-java:
	java -jar $(ANTLR4_JAR) TealangLexer.l4 -o gen/java
	java -jar $(ANTLR4_JAR) TealangParser.g4 -o gen/java
	javac gen/java/Tealang*.java -classpath "gen/java:$(ANTLR4_JAR)"

go: grammar-go
	go generate ./...
	go build -o tealang ./main.go

test:
	go test ./...

java-trace: grammar-java
	java -classpath "gen/java:$(ANTLR4_JAR)" org.antlr.v4.gui.TestRig Tealang program -diagnostics -trace $(ARGS)

java-gui: grammar-java
	java -classpath "gen/java:$(ANTLR4_JAR)" org.antlr.v4.gui.TestRig Tealang program -diagnostics -gui $(ARGS)

.PHONY: all test