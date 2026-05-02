TINYGO ?= tinygo

.PHONY: build build-std deploy dev clean

# Recommended: TinyGo produces a much smaller WASM binary
build:
	mkdir -p build
	$(TINYGO) build -o build/app.wasm -target wasm -no-debug .
	cp "$$($(TINYGO) env TINYGOROOT)/targets/wasm_exec.js" build/
	cp worker.mjs build/

# Alternative: standard Go (binary will be ~10-15 MB)
build-std:
	mkdir -p build
	GOOS=js GOARCH=wasm go build -o build/app.wasm .
	cp "$$(go env GOROOT)/misc/wasm/wasm_exec.js" build/
	cp worker.mjs build/

deploy: build
	wrangler deploy

dev: build
	wrangler dev

clean:
	rm -rf build
