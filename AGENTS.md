# Jarvis - Codex Instructions

## Project Overview

Jarvis is a Go-based local document chatbot and workbench with a Wails desktop shell and Alpine.js frontend. It uses Ollama for LLM chat, embeddings, and image understanding. Pure Go in-memory vector store. The app runs locally and is designed for privacy. It ships as a Windows NSIS installer and supports in-app auto-updates via GitHub Releases.

See [ROADMAP.md](ROADMAP.md) for planned features.

## Build and Run

```bash
go build ./cmd/server/
go run ./cmd/server/
go build -tags "desktop,production" ./cmd/desktop/
```

To build with a version baked in:

```bash
go build -tags "desktop,production" -ldflags="-X main.Version=v1.2.3" -o jarvis.exe ./cmd/desktop/
```

The server starts on port 8080 by default. Ollama must be running locally for development. The Windows installer bootstraps Ollama and the Jarvis models during manual installs.

Dependencies are resolved through Go modules. Do not commit `vendor/`; let CI download modules through `go mod download` or the normal `go` command.

## Versioning

Use semantic versioning for release tags: `vMAJOR.MINOR.PATCH`.

- Bump **major** for breaking changes: incompatible config changes, vector store format changes without migration, removed APIs, changed install locations, or changes that require manual user action.
- Bump **minor** for backward-compatible features: new loaders, new UI features, new API endpoints, new installer targets, new model options, or safe migrations.
- Bump **patch** for backward-compatible fixes: bug fixes, security fixes, log cleanup, small UI polish, CI fixes, packaging fixes, and documentation corrections.

The GitHub updater only runs on proper semver builds. When creating a release, bake the same tag into the binary with `-X main.Version=vMAJOR.MINOR.PATCH`.

## Writing Style

- Never use em dashes in code, comments, or documentation. Use commas, periods, or separate sentences instead.
- Write in plain, human language. Sound like a senior staff engineer explaining tradeoffs to a teammate.
- Prefer clear, boring code over clever code. Make names precise and keep control flow easy to follow.
- Keep comments short. One line max. Only add a comment when the reason behind the code is not obvious.

## Code Conventions

- Pure Go only. No CGo dependencies.
- Use `log/slog` for structured logging.
- Use Chi router for HTTP routing.
- All vector store operations must be thread-safe (use sync.RWMutex).
- Persist vector store with gob encoding, not SQLite.
- Frontend uses Alpine.js and Tailwind CSS from CDN. No build step.
- Large frontend files should be modularized as features grow. Prefer small focused JS modules or server-rendered partial templates before adding a build step.
- PowerShell scripts in CI must set `$ErrorActionPreference = "Stop"` at the top so failures are not silently swallowed.

## Architecture

- `cmd/server/main.go` is the browser/server entry point. `cmd/desktop/main.go` is the Wails desktop entry point. Both use `internal/app/runtime.go` for shared startup wiring. Each command declares `var Version = "dev"` which release builds overwrite via ldflags.
- `internal/app/` initializes config, Ollama, required models, vector store, document processor, RAG chain, research mode, task/chat stores, updater, and the shared HTTP server.
- `internal/config/` loads settings from environment variables. Key fields: `OllamaURL`, `OllamaKeepAlive`, `ChatModel`, `EmbeddingModel`, `VisionModel`, `GitHubRepo`, `GitHubToken`, `AppName`, `SupportEmail`, `SupportSubject`, and `SupportURL`.
- `internal/ollama/` wraps the Ollama Go client for embeddings, chat streaming, and image description.
- `internal/document/` handles file loading (PDF, DOCX, XLSX, PPTX, images, code, and unknown text-like files), URL fetching, text chunking, and the processing pipeline. Images are described by the vision model before embedding. Folder indexing skips unchanged files by comparing size and modification time. `web.go` handles HTML-to-text extraction for fetched URLs using `golang.org/x/net/html`.
- Clear indexed files must clear the vector store and remove only Jarvis-owned uploaded copies inside `DataDir`. It must not delete external source files that were indexed from a folder path, and it must preserve app state files such as chats, task state, and workspace maps.
- `internal/vectorstore/` is the in-memory vector store with hybrid retrieval, cosine vector similarity plus BM25 keyword scoring, and disk persistence.
- `internal/rag/` builds the RAG chain: embed the question, search for relevant chunks, build a prompt, and stream the response.
- `internal/server/` sets up the HTTP server, routes, and handlers. Keep route groups in focused files such as `chat_routes.go`, `document_routes.go`, `task_routes.go`, `system_routes.go`, and `update_routes.go`. Routes include `/api/v1/models` for installed Ollama models, plus `/api/v1/update` (GET) and `/api/v1/update/apply` (POST) for the GitHub updater.
- `internal/updater/` polls GitHub Releases, compares semver tags, and can download and silently launch the Windows NSIS installer. Release builds bake the GitHub repository into `builtinRepo` via ldflags so end users do not need repository configuration.
- `internal/websearch/` contains the DuckDuckGo scraper (`duckduckgo.go`) and the research orchestrator (`research.go`). The orchestrator uses `ChatOnce` to generate search queries without streaming, then emits structured progress events for the UI while it searches, fetches, and indexes pages.
- `internal/workbench/` contains the Codex-style memory layer: chat sessions, watched folders, workspace maps, symbol indexes, task messages, traces, and edit history persisted as JSON under the data directory. Workspace maps include regular documents, PDFs, images, spreadsheets, presentations, data files, source files, and code symbols.
- `web/` contains the single-page frontend (HTML template, JS, CSS). Keep Alpine app state and initialization in `web/static/js/app.js`, with feature behavior in focused files under `web/static/js/modules/`. Embedded into the binary via `go:embed` in `web/embed.go`, so the exe is self-contained.
- `web/static/js/modules/workbench.js` owns Workbench activity and workspace-map UI behavior.
- `packaging/windows/jarvis.nsi` is the NSIS installer script. Per-user install to `%LOCALAPPDATA%\Programs\Jarvis`, no UAC. Accepts `/S` for silent install. Manual installs run `packaging/windows/bootstrap-ollama.ps1`; silent auto-updates skip the bootstrap.
- `packaging/windows/bootstrap-ollama.ps1` checks for Ollama, installs it if missing, waits for the local API, and pulls `nomic-embed-text`, `gemma4:e2b`, `gemma4:e4b`, and `llava`.
- `packaging/linux/jarvis.desktop` is the Linux desktop entry bundled into the `.deb`/`.rpm` by GoReleaser.

## Release Pipeline

- `.github/workflows/ci.yml` builds/tests/vets on push and pull request, runs GoReleaser on semver tags with `.goreleaser.github.yaml`, and uploads the Windows NSIS installer to the GitHub Release.
- The workflow also builds `cmd/desktop` on `windows-latest` to catch Wails regressions on the free GitHub-hosted runner tier. Tagged Windows installers package the desktop binary as `jarvis.exe`; `cmd/server` remains a development and fallback target.
- GitHub Actions jobs must use standard GitHub-hosted runners such as `ubuntu-latest` and `windows-latest` so public repositories stay on the free runner tier.

## Key Design Decisions

- Ollama handles chat, embeddings (nomic-embed-text), and vision (llava). This avoids needing Python or PyTorch.
- The vector store is in-memory with gob persistence. This avoids CGo and works cleanly on Windows.
- RAG retrieval uses hybrid search. Keep BM25 pure Go and dependency-free so exact identifiers, error codes, function names, and config keys rank well alongside semantic matches.
- Folder ingest builds a lightweight workspace map for regular files and code. It classifies documents, PDFs, spreadsheets, presentations, images, data, config, text, and code files. It extracts symbols/imports only for supported text/code files. Keep scanners fast and dependency-free unless the roadmap explicitly moves to tree-sitter.
- Folder ingest persists watched folders. The app runs a background re-indexer that checks watched folders every 10 minutes, re-indexes only new or modified files, refreshes the workspace map, and records task traces when files change.
- The Workbench panel surfaces task traces, retrieval events, model state, workspace file type counts, searchable files, and searchable symbols. Keep it dense, practical, and work-focused.
- Reply-to-message and focused file mentions are part of the chat flow. Preserve `reply_to`, `focus_document_ids`, and `focus_files` support in `/api/v1/chat`.
- Task state and tool traces are persisted locally. Record chat requests, research events, retrieval summaries, future tool calls, and future edits there so sessions can be resumed after refresh.
- SSE (Server-Sent Events) is used for streaming chat responses. Simpler than WebSockets for this use case.
- The app falls back to general chat when no documents are indexed. It does not refuse to answer.
- Jarvis picks the chat model automatically when `CHAT_MODEL` is unset: use `gemma4:e2b` for very low-end machines and `gemma4:e4b` for everyone else. Do not auto-select `gemma4:26b`; it is opt-in through `CHAT_MODEL` only because Jarvis is local and RAG-first, and speed matters for daily use. Keep the main chat UI free of model-picker controls unless the roadmap explicitly reintroduces advanced model settings.
- Startup auto-pulls missing required Ollama models through the Go Ollama API as a fallback. Keep chat and embedding models required. Keep the vision model optional unless product requirements change. Do not remove startup verification just because the Windows installer bootstraps Ollama.
- Chat sessions are persisted server-side as JSON in the data directory and listed in the sidebar. Do not move primary chat history back to browser-only `localStorage`.
- Folder indexing walks directories recursively and skips common build output (bin, obj, node_modules, .git, vendor, etc).
- Vision model is optional. If not installed, the app still works but skips image processing.
- Images (standalone or extracted from PDFs) are described by the vision model. The description is embedded and stored like any other text chunk, making images searchable.
- Unknown extensions are accepted when the file looks like UTF-8 text. Binary files are rejected or skipped.
- PDF image extraction scans for embedded JPEG data in the PDF binary. Validated with `jpeg.DecodeConfig`. Non-JPEG images in PDFs are not extracted.
- Image byte slices are copied out of the PDF buffer and freed after vision processing. This prevents holding the entire PDF in memory.
- URL fetching extracts visible text from HTML pages. It skips script, style, nav, footer, and other non-content elements. Response bodies are limited to 10 MB.
- URLs pasted in the main chat box are auto-detected and fetched before the RAG query. URLs inside the code snippet box are treated as code and must not be fetched or indexed. Already-indexed URLs are skipped. Fetch failures are non-fatal.
- The chat header owns indexed files. The main input bar owns attachments: paper clip for files, folder icon for browser-selected folders, and a code snippet toggle with automatic language detection and syntax-highlighted preview. Avoid reintroducing large upload, folder path, URL fetch panels, or duplicate document buttons in the sidebar or composer.
- Pasted images in the chat input must show a compact preview before send. On send, upload them through the normal document pipeline so the vision model describes and indexes them before the chat request runs.
- The `web/` folder is embedded via `go:embed`. No external files are needed to run the exe.
- Research mode uses DuckDuckGo HTML scraping (no API key). The LLM generates 2-3 search queries, results are fetched and indexed, then the RAG chain answers with a research-specific prompt emphasising source citations. Research progress must be sent as structured SSE progress events, not markdown status text mixed into the answer.
- Prompt templates are client-side.
- The Wails desktop target starts a hidden loopback API server on `127.0.0.1` and injects that API base into the frontend. Keep this path for streaming endpoints because Wails' asset server can buffer response-body streaming on Windows.
- Build the Wails desktop target with `wails build` or `go build -tags "desktop,production" ./cmd/desktop`. A plain `go build ./cmd/desktop` binary shows Wails' missing build-tags error dialog.
- The auto-updater no-ops on `dev` builds. It only runs when `Version` is a proper semver string baked in at build time.
- Installer downloads use the `browser_download_url` from the latest GitHub Release asset. On Windows, Jarvis prefers an `.exe` asset whose name contains `setup`.
- Manual Windows installs should bootstrap Ollama and models. Silent installs should skip bootstrap so auto-updates stay fast and predictable.

## Memory and Performance Notes

- The processor copies image bytes out of source buffers and nils the `ImageData` field after vision processing. Memory usage stays proportional to one image at a time.
- Embedding happens in batches of 32 to balance throughput and memory.
- The vector store normalizes vectors on search, not on insert, so inserts are fast.
- Ollama swaps models in and out of GPU memory automatically. Switching between chat and vision models causes a brief pause for model loading.
