all: dist/solidifier dist/bridge

dist/solidifier: $(shell find solidifier/ -name "*.go")
	mkdir -p dist
	cd solidifier && go build -ldflags="-s -w" -o ../dist/solidifier

dist/bridge: $(shell find bridge/ -name "*.go")
	mkdir -p dist
	cd bridge && go build -ldflags="-s -w" -o ../dist/bridge
