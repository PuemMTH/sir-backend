# Project Binary Size Analysis (WASM)

This report provides a detailed breakdown of the package size contributions to the `app.wasm` binary (~14.2 MB).

## 1. Top Package Size Contributions (Estimated from Archives)

The following table lists the largest package archives (`.a` files) generated during the build process. Note that archive size includes metadata and is generally larger than the final linked size, but it serves as a good proxy for relative contribution.

| Package Name | Build ID | Size (Bytes) | Size (MB) | Description |
| :--- | :--- | :--- | :--- | :--- |
| **runtime** | `b010` | 9,387,888 | 8.95 MB | Go Runtime (Garbage Collector, Scheduler, etc.) |
| **net/http** | `b114` | 7,466,910 | 7.12 MB | Standard HTTP client/server |
| **crypto/tls** | `b122` | 4,066,436 | 3.88 MB | TLS implementation (required by net/http) |
| **net** | `b168` | 3,654,262 | 3.48 MB | Network primitives |
| **reflect** | `b050` | 2,594,070 | 2.47 MB | Reflection (heavily used by GORM/JSON) |
| **gorm.io/gorm** | `b195` | 2,579,124 | 2.46 MB | GORM Main Package |
| **math/big** | `b105` | 1,488,660 | 1.42 MB | Big integers (used by crypto) |
| **os** | `b052` | 1,459,608 | 1.39 MB | Operating System primitives |
| **crypto/x509** | `b162` | 1,810,454 | 1.73 MB | Certificate management |
| **encoding/json** | `b002` | 1,673,710 | 1.60 MB | JSON encoding/decoding |

## 2. Dependency Groups

The large binary size is primarily driven by three major dependency "gravity wells":

### A. The Go Runtime (~60% of total)
The standard Go runtime is not optimized for WASM/Cloudflare Workers. It includes a full garbage collector and scheduler that must be embedded in the binary.
*   **Key Packages:** `runtime`, `reflect`, `internal/abi`.

### B. Network & Security (~25% of total)
Using `net/http` for simple API routing pulls in a massive chain of dependencies, including the entire TLS stack and X.509 certificate management, even if you are only handling incoming requests on Cloudflare (which already handles TLS).
*   **Key Packages:** `net/http`, `crypto/tls`, `crypto/x509`, `net`.

### C. GORM & Reflection (~10% of total)
GORM relies heavily on the `reflect` package to map database results to Go structs. This not only adds the GORM library itself but also ensures that the linker cannot "tree-shake" (remove) many parts of the reflection and type system.
*   **Key Packages:** `gorm.io/gorm`, `reflect`.

## 3. Strategic Recommendations

To significantly reduce the binary size (from 14MB down to <1MB):

1.  **Use TinyGo:** TinyGo provides an alternative runtime specifically designed for small binaries. It replaces the heavy Go runtime with a minimal one.
2.  **Avoid `net/http` for routing:** Use simpler routers that don't depend on the full `net/http` stack, or use the native `syumai/workers` request handling.
3.  **Replace GORM with Raw SQL:** Using `database/sql` or the D1 driver directly avoids the overhead of reflection and the large GORM codebase.
4.  **Use `wasm-opt`:** Post-processing the WASM file with `wasm-opt -Oz` can further compress the binary by another 10-30%.
