# Cognitive Package Manager (cpm)

`cpm` is the CognitiveOS package manager — installs, updates, and removes `.cgp` cognitive patches.

## Build

```bash
go build -o bin/cpm ./cmd/cpm
```

## Commands

- `cpm install <name|path>` — install a `.cgp` patch
- `cpm remove <name>` — uninstall
- `cpm list` — list installed patches
- `cpm info <name>` — show manifest details
- `cpm search <query>` — search registry

## Architecture

- Go CLI binary
- Reads `cognitive.json` manifest from `.cgp` archives
- Performs hardware audit before install (RAM, storage, NPU)
- Spawns MCP servers declared in manifest as subprocesses
- Installs to `/cognitiveos/patches/<name>/`

## Dependencies

- Uses product-specs for `.cgp` format and `cognitive.json` schema
- Communicates with registry-server for remote installs
