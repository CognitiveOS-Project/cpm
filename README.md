# cpm — Cognitive Package Manager

`cpm` installs, updates, removes, and publishes `.cgp` (Cognitive Patch) files for CognitiveOS.

It is the npm/pip/apt for the agent era — hardware-aware, MCP-native, and designed for autonomous AI operation.

## Quick Start

```bash
go build -o bin/cpm ./cmd/cpm

# Create a skill skeleton
./bin/cpm init my-skill
cd my-skill
# Edit cognitive.json, add prompts/ and tools/

# Install from a local archive
./bin/cpm install ./my-skill.cgp

# List installed patches
./bin/cpm list

# Show patch details
./bin/cpm info my-skill

# Remove a patch
./bin/cpm remove my-skill
```

## Commands

| Command | Description |
|---------|-------------|
| `init <dir>` | Create a .cgp skeleton directory |
| `install <path\|name>` | Install from local .cgp or registry |
| `remove <name>` | Uninstall a patch |
| `list` | List installed patches |
| `info <name>` | Show manifest details |
| `verify <path>` | Validate a .cgp archive |
| `search <query>` | Search the registry |
| `update <name>` | Update to latest version |

## Development

```bash
export CPM_PATCHES_DIR=/tmp/cpm-test/patches
export CPM_CACHE_DIR=/tmp/cpm-test/cache
```

<<<<<<< Updated upstream
## Related

- [CognitiveOS](https://github.com/CognitiveOS-Project/cognitiveos) — main project repository
- [cognitive-os.org](https://cognitive-os.org) — project website
- [Registry Server](https://github.com/CognitiveOS-Project/registry-server) — .cgp package registry
- [cgp-template](https://github.com/CognitiveOS-Project/cgp-template) — .cgp package boilerplate
- [Product Specs](https://github.com/CognitiveOS-Project/product-specs) — .cgp format specification
- [CognitiveOS Project](https://github.com/CognitiveOS-Project) — GitHub organization
=======
## Contributing

1. Branch from `development`, not `main`
2. Use topic branches: `feature/<name>`, `fix/<name>`, `bugfix/<name>`
3. Open a PR to `development` with a clear title and description
4. Merge via squash after review
5. Changes flow to `main` via a release PR

See the [SDLC repo](https://github.com/CognitiveOS-Project/sdlc) for the full contribution guide, code review standards, and testing strategy.
>>>>>>> Stashed changes

## Author

**Jean Machuca** — [GitHub](https://github.com/jeanmachuca) · [Sponsor](https://github.com/sponsors/jeanmachuca)

## License

MIT
