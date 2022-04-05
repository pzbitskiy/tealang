default: build

all: go test grammar-java
build: go

GO_ALGORAND := github.com/algorand/go-algorand

algorand-install:
	VERSION=$$(grep -F '$(GO_ALGORAND)' go.mod | cut -d " " -f2) && \
	REVISION=$$(grep -F '$(GO_ALGORAND)' go.mod | cut -d " " -f2 | cut -d "-" -f3) && \
	MOD_LOC=$$(go env GOPATH)/pkg/mod/$(GO_ALGORAND)@$$VERSION && \
	[ ! -d "$$MOD_LOC" ] && go get -u github.com/algorand/go-algorand@$$REVISION || true

	VERSION=$$(grep -F '$(GO_ALGORAND)' go.mod | cut -d " " -f2) && \
	REVISION=$$(grep -F '$(GO_ALGORAND)' go.mod | cut -d " " -f2 | cut -d "-" -f3) && \
	MOD_LOC=$$(go env GOPATH)/pkg/mod/$(GO_ALGORAND)@$$VERSION && \
	chmod -R 755 $$MOD_LOC && \
	cd $$MOD_LOC && \
	mkdir -p scripts/buildtools && \
	curl -o scripts/buildtools/install_buildtools.sh https://raw.githubusercontent.com/algorand/go-algorand/$$REVISION/scripts/buildtools/install_buildtools.sh && \
	curl -o scripts/buildtools/go.mod https://raw.githubusercontent.com/algorand/go-algorand/$$REVISION/scripts/buildtools/go.mod && \
	chmod +x scripts/buildtools/install_buildtools.sh && \
	./scripts/configure_dev.sh && ./scripts/buildtools/install_buildtools.sh && make build && \
	cd -

ANTLR4_VER := 4.9.3
ANTLR4_JAR := /usr/local/lib/antlr-$(ANTLR4_VER)-complete.jar

antlr-install:
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