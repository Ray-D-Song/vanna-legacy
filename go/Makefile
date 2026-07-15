BINARY      ?= vanna
DEMO_BINARY ?= vanna-demo
CONFIG      ?= config.yaml
GO          ?= go

.PHONY: dev build build-demo clean

dev:
	$(GO) run ./cmd/vanna serve --config $(CONFIG)

build:
	$(GO) build -o $(BINARY) ./cmd/vanna

build-demo:
	$(GO) build -o $(DEMO_BINARY) ./cmd/demo
	./$(DEMO_BINARY)

clean:
	rm -f $(BINARY) $(DEMO_BINARY)
