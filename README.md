# cpm — Cognitive Package Manager

`cpm` installs, updates, removes, and publishes `.cgp` (Cognitive Patch) files for CognitiveOS.

It is the npm/pip/apt for the agent era — hardware-aware, MCP-native, and designed for autonomous AI operation.

## Quick Start

```bash
make build

# Create a skill skeleton
./build/bin/cpm init my-skill
cd my-skill
# Edit cognitive.json, add prompts/ and tools/

# Pack into a .cgp archive
./build/bin/cpm pack

# Search the registry
./build/bin/cpm search skill

# Install from a local archive
./build/bin/cpm install ./my-skill.cgp

# Install from the registry (with optional unlock)
./build/bin/cpm install my-skill
./build/bin/cpm install my-skill --unlock <code>

# List installed patches
./build/bin/cpm list

# Show patch details
./build/bin/cpm info my-skill
./build/bin/cpm info my-skill --json

# Remove a patch
./build/bin/cpm remove my-skill
```

## Commands

| Command | Description |
|---------|-------------|
| `cpm init <dir>` | Create a .cgp skeleton directory |
| `cpm install <path\|name>` | Install from local .cgp or registry |
| `cpm install <name> --unlock <code>` | Install a locked package with an unlock code |
| `cpm remove <name>` | Uninstall a patch |
| `cpm list` | List installed patches |
| `cpm info <name>` | Show manifest details |
| `cpm info <name> --json` | Machine-readable JSON output |
| `cpm verify <path>` | Validate a .cgp archive |
| `cpm pack` | Create a .cgp archive from current directory |
| `cpm search <query>` | Search the registry |
| `cpm update <name>` | Update to latest version |
| `cpm tune` | Tune inference parameters |
| `cpm publish <path>` | Publish a package to the registry |
| `cpm download-weights <name>` | Download model weights for a package |

### Auth Commands

| Command | Description |
|---------|-------------|
| `cpm auth signup --key <path>` | Register machine identity profile with the registry |
| `cpm auth login --key <path>` | Store SSH key path locally and verify with registry |
| `cpm auth logout` | Clear local auth state |

### Publish

Publish a package to the registry:

```bash
# Official publish — server creates GitHub Release and hosts the .cgp
cpm publish ./my-skill.cgp

# Notary proxy — register metadata only, host the .cgp yourself
cpm publish ./my-skill.cgp --download-url https://example.com/my-skill.cgp
```

**Key resolution** (in order): `--key` flag → `~/.cpm/auth.json` (from `cpm auth login`) → default `~/.ssh/id_ed25519` → `CPM_REGISTRY_TOKEN` (deprecated).

**32 MB size guard:** Official publish rejects `.cgp` files over 32 MB (Cloud Run limit). For large packages, use `--download-url` to host externally, or split weights from the core archive.

### Init CI/CD Templates

`cpm init` generates `.github/docker/Dockerfile.ci`, `.github/workflows/ci.yml`, and `.github/workflows/publish.yml` alongside the skeleton:

- **Dockerfile.ci:** Alpine multi-stage build with `cpm pack` + `cpm info --json`
- **CI workflow:** Builds via Docker Buildx, extracts artifacts, outputs package metadata
- **Publish workflow:** Same build, then creates a GitHub Release with the `.cgp` asset

## Development

```bash
export CPM_PATCHES_DIR=/tmp/cpm-test/patches
export CPM_CACHE_DIR=/tmp/cpm-test/cache
```

## Related

- [CognitiveOS](https://github.com/CognitiveOS-Project/cognitiveos) — main project repository
- [cognitive-os.org](https://cognitive-os.org) — project website
- [Registry Server](https://github.com/CognitiveOS-Project/registry-server) — .cgp package registry
- [cgp-template](https://github.com/CognitiveOS-Project/cgp-template) — .cgp package boilerplate
- [coginit](https://github.com/CognitiveOS-Project/coginit) — boot manager that orchestrates CognitiveOS services
- [Product Specs](https://github.com/CognitiveOS-Project/product-specs) — .cgp format specification
- [CognitiveOS Project](https://github.com/CognitiveOS-Project) — GitHub organization

## Build

```bash
make build    # Compile to build/bin/cpm
make test     # Run tests
make lint     # Run go vet
make clean    # Remove build artifacts
```

## Contributing

1. Branch from `development`, not `main`
2. Use topic branches: `feature/<name>`, `fix/<name>`, `bugfix/<name>`
3. Open a PR to `development` with a clear title and description
4. Merge via squash after review
5. Changes flow to `main` via a release PR

See the [SDLC repo](https://github.com/CognitiveOS-Project/sdlc) for the full contribution guide, code review standards, and testing strategy.

## Author

**Jean Machuca** — [GitHub](https://github.com/jeanmachuca) · [Sponsor](https://github.com/sponsors/jeanmachuca)

## License

MIT
