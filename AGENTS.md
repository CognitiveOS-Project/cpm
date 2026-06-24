# Cognitive Package Manager (cpm)

`cpm` installs, removes, and manages `.cgp` cognitive patches for CognitiveOS.

## Build

```bash
export PATH="/tmp/go/bin:$(go env GOPATH)/bin:$PATH"
go build -o bin/cpm ./cmd/cpm
```

## Commands

| Command | Description | Status |
|---------|-------------|--------|
| `cpm init <dir>` | Create .cgp skeleton | ✅ |
| `cpm install <path\|name>` | Install from local file or registry | ✅ |
| `cpm remove <name>` | Uninstall | ✅ |
| `cpm list` | List installed | ✅ |
| `cpm info <name>` | Show manifest | ✅ |
| `cpm verify <path>` | Validate archive | ✅ |
| `cpm search <query>` | Search registry | ✅ |
| `cpm update <name>` | Update to latest | ✅ |

## Development

Set env vars for testing without `/cognitiveos`:
```bash
export CPM_PATCHES_DIR=/tmp/cpm-test/patches
export CPM_CACHE_DIR=/tmp/cpm-test/cache
```

## Architecture

- **cmd/** — Cobra CLI commands
- **internal/archive/** — .cgp tar.gz parsing/extraction
- **internal/audit/** — Hardware resource checking
- **internal/config/** — registries.toml parsing
- **internal/dep/** — Dependency resolution
- **internal/log/** — Structured logging
- **internal/patch/** — Patch lifecycle (install/remove/list)
- **internal/registry/** — HTTP client for registry-server
- **internal/schema/** — JSON Schema validation (embedded schema)

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/santhosh-tekuri/jsonschema/v6` — JSON Schema validation
- `gopkg.in/ini.v1` — TOML config parsing
