# Deployment Guide: Cloudflare Worker (Go)

This document outlines the steps to build and deploy this Go-based Cloudflare Worker. The project has been configured to use the standard Go WebAssembly (Wasm) runtime, which natively handles fetch event bridging via `github.com/syumai/workers`.

## Prerequisites
- **Go** (v1.21 - v1.23 recommended)
- **Node.js & npm** (Required to execute Cloudflare's Wrangler CLI)

## Deployment Command

To deploy the worker to Cloudflare, simply run the following command in the root of your project:

```sh
npm_config_ignore_scripts=true npx --yes wrangler deploy --config wrangler.toml
```

### What happens under the hood?
1. **`npm_config_ignore_scripts=true`**: This environment variable disables npm post-install scripts. This is required because modern Node.js versions (like Node 25) can fail to build local binaries (such as `sharp`) when fetching Wrangler dependencies.
2. **`npx --yes wrangler deploy`**: Downloads and executes the Cloudflare Wrangler CLI.
3. **`make build-std`**: Wrangler looks at `wrangler.toml` and automatically triggers the standard Go build process.
4. **`workers-assets-gen`**: The Makefile automatically fetches the official Cloudflare Wasm worker shims and builds `app.wasm` fitting well within the Cloudflare Worker size limits.

## Local Development / Testing

If you want to test the worker locally on your machine before pushing to Cloudflare, use the `dev` command:

```sh
npm_config_ignore_scripts=true npx --yes wrangler dev --config wrangler.toml
```

This will start a local server where you can test your APIs out (e.g. `http://localhost:8787/hello`).