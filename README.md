# Jarvis - Chat with your Documents

A local, privacy-first document chatbot and workbench built with Go, Wails, and Alpine.js. Upload PDFs, DOCX, XLSX, PPTX, images, code files, or point it at an entire folder. It indexes everything locally and lets you ask questions using a local Gemma-first LLM through Ollama today, with direct llama.cpp support planned. It can also search the web and research topics for you.

No data leaves your machine. No API keys needed. Single binary, runs anywhere.

## Features

- **Chat with documents:** upload files or index entire folders, then ask questions with source citations
- **Natural context routing:** generic questions stay general even when files are indexed. Jarvis uses indexed context when you ask about documents, name a file, or focus a file with `@` or `#`
- **Reply context:** reply to a specific prior message so follow-up questions carry the intended local context
- **Focused file mentions:** type or select `@file` and `#file` mentions so retrieval prioritizes specific indexed files
- **Message queue:** queue follow-up prompts while an answer is streaming, edit queued prompts inline, reorder them, or remove them before they run
- **Response controls:** copy answers, rate responses, fork a conversation from a response, and see live/final response timing
- **General chat:** works as a regular assistant even without documents loaded
- **Streaming responses:** token-by-token SSE streaming with markdown rendering
- **Recursive folder indexing:** use the sidebar folder button to point Jarvis at a project or regular folder. It walks subdirectories while skipping build output and Jarvis-owned app state
- **Watched folders:** indexed folders are remembered and checked in the background, so new and changed files are refreshed without clearing the index
- **Wide file support:** PDF, DOCX, XLSX, PPTX, images, known source files, and unknown text-like files. Binary files are rejected
- **Hybrid retrieval:** combines vector similarity with BM25 keyword scoring for better exact matches on code symbols, error codes, and config keys
- **Workspace map:** folder ingest maps regular files, Office documents, PDFs, images with dimensions, data files, source files, code symbols, packages, tests, and folder groups
- **Project foundation:** indexed folders are saved as projects with an active project, workspace map, watched re-indexing, and a project vector store for folder-scoped retrieval
- **Workbench activity:** inspect the active project, workspace map summary, local git changes, and recent task activity without crowding the chat UI
- **Read-only agent tools:** normal chat can let the model request safe project search, file summaries, and file previews before answering
- **Image support:** upload standalone images (PNG, JPG, etc.) or PDFs with embedded images. A vision model describes each image so it becomes searchable and queryable
- **Gemma 4 controls:** choose Gemma 4 2B, 4B, or 26B from the composer dropdown, and enable optional thinking mode when deeper reasoning helps
- **Live context:** normal chat uses current date/time, browser locale, timezone, pasted URLs, and selective web context for current questions when available, without searching every message
- **Deep research mode:** toggle research mode from the composer dropdown when you want visible search progress, local date/time and locale-aware queries, fetched sources, citations, and links. No API key needed
- **URL fetching:** paste a website link directly in chat. Jarvis auto-detects URLs in normal chat messages, fetches the page, and indexes it quietly. URLs inside the code snippet box are treated as code and are not fetched
- **Regional awareness:** automatically detects your locale and timezone from the browser. Answers use your local currency, date formats, and regionally relevant context
- **Document management:** upload, list, delete individual docs, or clear everything at once. Clearing indexed files removes Jarvis-owned upload copies and preserves external source files
- **Server-side chat history:** chats are saved locally under the data directory and shown in the sidebar
- **Codex-style desktop UI:** neutral dark workbench theme with a focused chat surface, consistent panels, compact controls, and responsive layout
- **Chat-bar attachments** with file picker, pasted image support, code snippets, and progress tracking. Folder indexing lives in the sidebar
- **Code snippet mode** with automatic language detection and syntax-highlighted preview
- **Self-contained binary:** the web UI is embedded into the exe via go:embed. No external files needed. Pure Go, no CGo, builds on Windows/Linux/Mac

## Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Ollama](https://ollama.com/) running locally

On Windows, the NSIS installer bootstraps Ollama for manual installs. It checks for `ollama.exe`, installs Ollama if missing, starts it, and pulls the Jarvis models. Silent auto-updates skip this bootstrap step so updates stay fast.

For development or manual setup, pull the required models:

```bash
ollama pull gemma4:e2b
ollama pull gemma4:e4b
ollama pull nomic-embed-text
ollama pull llava              # optional, enables image support
```

Jarvis picks the chat model automatically when `CHAT_MODEL` is not set. Very low-end machines use `gemma4:e2b`; everyone else uses `gemma4:e4b`. The composer dropdown can switch a request between Gemma 4 2B, 4B, and 26B. If the selected dropdown model is missing, Jarvis asks Ollama to download it before answering. The larger `26b` model is not selected automatically because Jarvis is local and RAG-first, and speed matters more for daily use. If Ollama is running but the required chat or embedding model is missing, Jarvis downloads it automatically on startup. Manual pulls are still useful for preparing a machine ahead of time.

Gemma 4 is treated as the primary model family. Jarvis uses Gemma 4 defaults for chat quality, including `temperature=1.0`, `top_p=0.95`, and `top_k=64`. Query generation stays deterministic. Based on Google's official Gemma 4 model card and the local runtime guide, Jarvis treats Gemma as text-output multimodal understanding, not native media generation: E2B/E4B can handle text, image, audio, and short video understanding, while 26B-A4B/31B focus on stronger text and image reasoning. The optional thinking mode adds Gemma's `<|think|>` control token to the system prompt and Jarvis strips thought blocks before saving chat history. Image, audio, or video generation would need separate optional generation models.

## Quick Start

```bash
# Clone and enter the project
cd Pdf_Chatbot

# Build
go build -o server.exe ./cmd/server/

# Run
./server.exe
```

Open [http://localhost:8080](http://localhost:8080) in your browser.

To build the desktop shell:

```bash
go build -tags "desktop,production" -o jarvis-desktop.exe ./cmd/desktop/
```

The desktop app uses Wails with the same embedded frontend and Go backend. It is the primary end-user app. The browser server remains available for development and fallback use.

## Release Versioning

Jarvis uses semantic versioning for release tags: `vMAJOR.MINOR.PATCH`, for example `v1.4.2`. The GitHub updater only runs on proper semver builds, so release tags should always use this format.

- **Major**: increment for breaking changes. Examples: incompatible config changes, vector store format changes without migration, removed APIs, changed install locations, or behavior that requires users to manually reconfigure Jarvis.
- **Minor**: increment for backward-compatible features. Examples: new document loaders, new UI features, new API endpoints, new installer targets, new model options, or safe storage migrations.
- **Patch**: increment for backward-compatible fixes. Examples: bug fixes, security fixes, log noise cleanup, small UI polish, CI fixes, packaging fixes, and documentation corrections.

When creating a release, build with the same version baked into the binary:

```bash
go build -tags "desktop,production" -ldflags="-X main.Version=v1.4.2" -o jarvis.exe ./cmd/desktop/
```

### Recommended v2.1.0 release

If the previous public release was `v2.0.1`, the next tag should be `v2.1.0`. This release adds backward-compatible features across the desktop UI, document indexing, workspace mapping, chat context, installer bootstrap, and GitHub updater flow.

Suggested highlights:

- Wails desktop app is the primary installed experience
- GitHub Releases based updater and Windows NSIS installer flow
- Windows installer bootstrap for Ollama and required Jarvis models
- Reply-to-message context and `@file` / `#file` focused retrieval
- Queued follow-up prompts with edit, reorder, and remove controls
- Copy, rate, fork, and response timing controls
- Workspace map for regular folders, Office files, PDFs, image dimensions, data, text, config, code symbols, packages, tests, and folder groups
- Watched folder background re-indexing for new and modified files
- Simplified Workbench panel for project summary, workspace map counts, local changes, and recent activity
- Local changes panel with changed-file totals and expandable git diffs for the active project
- Agent foundations for project-scoped folder indexes, model-planned read-only project tools, and approved command execution, kept out of the default Workbench surface
- Safer clear-index behavior that preserves external source files
- Neutral Codex-style UI theme and expanded in-app Help guide

## Configuration

All settings are configurable through environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `OLLAMA_URL` | `http://localhost:11434` | Ollama API endpoint |
| `OLLAMA_KEEP_ALIVE` | `120s` | How long Ollama keeps models loaded after a request. Use `0s` for lowest idle memory |
| `CHAT_MODEL` | auto | LLM model for chat. Leave unset for hardware-aware selection, or set it to force a model |
| `EMBEDDING_MODEL` | `nomic-embed-text` | Model for generating embeddings |
| `VISION_MODEL` | `llava` | Vision model for describing images (optional) |
| `DATA_DIR` | `./data` | Directory for uploaded files |
| `VECTORSTORE_DIR` | `./vectorstore` | Directory for persisted embeddings |
| `CHUNK_SIZE` | `1000` | Characters per text chunk |
| `CHUNK_OVERLAP` | `200` | Overlap between chunks |
| `TOP_K` | `5` | Number of relevant chunks to retrieve |
| `MAX_UPLOAD_MB` | `50` | Maximum upload file size in MB |
| `GITHUB_REPO` | empty | GitHub repository for update checks, in `owner/repo` format. Release builds can bake this in automatically |
| `GITHUB_TOKEN` | empty | Optional GitHub token for private release checks and downloads |
| `APP_NAME` | `Jarvis` | Display name used in the UI and exported chats |
| `SUPPORT_EMAIL` | empty | Optional support email shown in the UI |
| `SUPPORT_SUBJECT` | `Jarvis Support` | Subject used for mail support links |
| `SUPPORT_URL` | empty | Optional support URL. Takes precedence over `SUPPORT_EMAIL` |

Example with custom settings:

```bash
CHAT_MODEL=gemma4:31b PORT=3000 ./server.exe
```

## Project Structure

```
cmd/server/main.go            Browser/server entry point
cmd/desktop/main.go           Wails desktop entry point
internal/
  app/runtime.go               Shared application bootstrap for server and desktop modes
  config/config.go             Environment-based configuration
  ollama/client.go             Ollama API wrapper (chat, embeddings, vision)
  gemma/                       Gemma 4 profiles, capabilities, options, thinking cleanup
  websearch/
    duckduckgo.go              DuckDuckGo HTML search scraper
    research.go                Research orchestrator (query gen, search, fetch)
  document/
    loader.go                  File loader interface and registry
    pdf.go                     PDF text and image extraction
    docx.go                    DOCX text extraction
    xlsx.go                    XLSX text extraction
    pptx.go                    PowerPoint text extraction
    image.go                   Standalone image loader
    text.go                    Plain text, source code, and unknown text-like file loader
    web.go                     URL fetching and HTML text extraction
    chunker.go                 Recursive text splitter
    processor.go               Orchestrates load, chunk, embed, store
  vectorstore/
    store.go                   In-memory vector store with hybrid retrieval
    persistence.go             Save/load embeddings to disk (gob format)
    math.go                    Vector math utilities
  workbench/
    chat.go                    Local chat session store
    folders.go                 Watched folder persistence for background re-indexing
    repo.go                    Workspace map, file classification, package grouping, test detection, and symbol scanning
    task.go                    Task state, traces, and edit history
  rag/
    chain.go                   RAG pipeline: retrieve, prompt, stream
    prompt.go                  System prompt templates
  server/
    server.go                  HTTP server with Chi router
    routes.go                  Route registration
    chat_routes.go             Chat and research streaming handlers
    chat_session_routes.go     Local chat session handlers
    document_routes.go         Upload, ingest, document, and URL handlers
    task_routes.go             Task state and trace handlers
    system_routes.go           Health, config, system, and model handlers
    update_routes.go           GitHub updater handlers
web/
  templates/index.html         Frontend (Alpine.js + Tailwind)
  static/js/app.js             Alpine app state and initialization
  static/js/modules/           Focused browser-loaded frontend modules
    message_*.js               Chat composer, streaming, queue, context, attachments, and response actions
  static/css/app.css           Custom styles
  embed.go                     Embeds web/ into the binary
packaging/
  windows/jarvis.nsi           Windows NSIS installer
  windows/bootstrap-ollama.ps1 Ollama and model bootstrap for manual Windows installs
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET /` | Serve the web UI |
| `POST /api/v1/chat` | Send a message (SSE streaming response) |
| `GET /api/v1/chats` | List locally saved chat sessions |
| `POST /api/v1/chats` | Create a new chat session |
| `GET /api/v1/chats/{id}` | Load a chat session |
| `PUT /api/v1/chats/{id}` | Save messages for a chat session |
| `DELETE /api/v1/chats/{id}` | Delete a chat session |
| `POST /api/v1/upload` | Upload and index a file |
| `POST /api/v1/fetch-url` | Fetch a webpage and index its text |
| `POST /api/v1/ingest` | Index all files in a folder path |
| `GET /api/v1/documents` | List indexed documents |
| `DELETE /api/v1/documents/{id}` | Remove a specific document |
| `DELETE /api/v1/documents` | Clear all documents and embeddings |
| `GET /api/v1/config` | Runtime UI config such as app name and support contact |
| `GET /api/v1/health` | Health check with model and store info |
| `GET /api/v1/system` | Hardware and system status |
| `GET /api/v1/models` | List installed Ollama models and the default chat model |
| `GET /api/v1/projects` | List persisted projects and the active project |
| `POST /api/v1/projects` | Open a folder as the active project |
| `GET /api/v1/diff` | Changed-file summary and text patches for the active git project |
| `GET /api/v1/repo-map` | Current workspace map with files, file kinds, imports, symbols, image dimensions, packages, tests, and folder groups |
| `GET /api/v1/tools/files` | Search active-project files by path, kind, language, or imports |
| `POST /api/v1/tools/read-file` | Safely read a text file inside the active project |
| `POST /api/v1/tools/summarize-file` | Return metadata, symbols, imports, and a short excerpt for one active-project file |
| `POST /api/v1/tools/run-command` | Run an approved allowlisted command in the active project and capture output |
| `GET /api/v1/task` | Current persisted task state, messages, traces, and edit history |
| `POST /api/v1/task/traces` | Append a tool trace event to the current task |
| `POST /api/v1/task/edits` | Append an edit-history entry to the current task |
| `DELETE /api/v1/task` | Clear current task state |

## How It Works

1. **Upload or index:** files are split into overlapping text chunks. Images (standalone or extracted from PDFs) are described by a vision model, and those descriptions become searchable text. URLs pasted in the main chat box are auto-detected and fetched. Indexed folders are remembered for background refreshes.
2. **Embed:** each chunk is converted to a vector using `nomic-embed-text` via Ollama
3. **Store:** vectors are kept in memory and persisted to disk in gob format
4. **Map:** folder ingest also builds a workspace map for files, documents, image dimensions, data, code symbols, imports, packages, tests, and folder groups
5. **Refresh:** watched folders are checked in the background. New and modified files are re-indexed, and the workspace map is refreshed.
6. **Query:** your question is embedded, the most similar chunks are retrieved, and they are passed as context to the LLM. Reply context and focused `@file` mentions are included when present. Your locale, timezone, and current local date/time are included so answers use local conventions.
7. **Stream:** the LLM response streams back token-by-token via Server-Sent Events. While it is streaming, you can queue, edit, reorder, or remove follow-up prompts.
8. **Act:** after a response, copy it, rate it, fork a new conversation from that point, or inspect how long generation took.

### Workbench

The Workbench panel surfaces local task and workspace state:

- task traces from chat, research, retrieval, workspace map updates, and future tools
- active project metadata and per-project vector-store path
- selected model, message count, trace count, and edit count
- workspace map root, file count, symbol count, package count, test count, folder groups, and file type breakdown
- local git change summary with per-file additions, deletions, status, and expandable text patches
- recent task activity

Jarvis hides its own app-state JSON, such as chat history, task state, project metadata, and workspace-map files, from user-facing sources. Those files remain available to the app internally but should not appear as evidence for normal answers.

This is the bridge from document chat toward a local coding and knowledge workbench while keeping all state local.

### Agent workflow direction

Jarvis is being prepared for a Codex-like local agent workflow. The current release adds the project foundation: indexed folders become active projects with workspace maps, watched re-indexing, and a per-project vector store for folder-scoped retrieval.

The next layers are:

- project-scoped vector stores are now used for folder indexes. New chats are tagged to the active project, while legacy global chats remain visible. Approved command policy, command history, edit history, and patch state are stored per project.
- read-only tools for listing, searching, reading, and summarizing project files. Normal chat can now ask the model for a bounded JSON tool plan, run safe active-project file tools, trace the calls, and fall back to deterministic file summaries when planning fails.
- safe command execution with approvals, timeouts, and captured output. The first allowlisted command runner is available in the Workbench.
- patch generation, review, apply, and discard flows. The current Workbench already shows read-only local diffs for the active git project.
- test and fix loops that keep all changes visible and reversible

### Chat workflow

The chat UI supports a few context controls that make local models more useful:

- reply to a message to anchor a follow-up to that exact turn
- type `@` or `#` to focus retrieval on a specific indexed file
- paste images into the composer so they are indexed before the question runs
- ask current questions naturally. Normal chat can quietly refresh live web context when the question is clearly time-sensitive
- queue follow-up messages while the current response streams
- edit or reorder queued follow-ups before Jarvis sends them
- fork a conversation from an assistant response when you want a new branch of thought

### Research mode

Toggle the magnifying glass button next to the chat input. When active:

1. The LLM generates 2-3 focused search queries from your question
2. Query generation includes your browser locale, timezone, local date, and local time for regional questions like currency, weather, pricing, and local rules
3. Each query is searched on DuckDuckGo (no API key needed)
4. The top results are fetched and indexed through the same pipeline as uploaded files
5. The LLM then answers using the fetched content, with citations and source links
6. Fetched articles stay in your vector store, so follow-up questions reuse them without re-fetching

Research steps stream as a compact progress timeline while the answer is being prepared.

## Hardware Requirements

Ollama runs the models on your hardware. Jarvis itself uses very little memory, typically under 100 MB even with thousands of indexed chunks. The models are what need the horsepower.

### Model resource usage

| Model | Purpose | VRAM (GPU) | RAM (CPU only) |
|-------|---------|-----------|----------------|
| `gemma4:e4b` | Chat (lighter) | ~10 GB | ~12-16 GB |
| `gemma4:e2b` | Chat (very light) | ~7 GB | ~8-12 GB |
| `nomic-embed-text` | Embeddings | ~0.3 GB | ~0.5 GB |
| `llava` (7b) | Vision (optional) | ~5 GB | ~6-8 GB |

Ollama swaps models in and out of GPU memory automatically. Only one model is loaded at a time, so your peak usage equals the largest model being used at that moment.

### Recommended setups

**With a GPU (much faster):**

| GPU VRAM | Recommended config |
|----------|-------------------|
| Under 8 GB | `gemma4:e2b` + `nomic-embed-text` |
| 8 GB+ | `gemma4:e4b` + `nomic-embed-text` |
| 16 GB+ | `gemma4:e4b` + `nomic-embed-text` + `llava` |

**CPU only (no GPU):**

| RAM | What to expect |
|-----|---------------|
| Under 12 GB | Uses `gemma4:e2b`. Good for basic chat and simple document Q&A |
| 12 GB+ | Uses `gemma4:e4b`. Good general default for most CPU-only machines |
| 32 GB+ | Still uses `gemma4:e4b` automatically. Larger models can be forced if you accept latency |

To force a specific chat model:

```bash
ollama pull gemma4:e4b
CHAT_MODEL=gemma4:e4b ./server.exe
```

For a faster coding-focused model:

```bash
ollama pull qwen2.5-coder:14b
CHAT_MODEL=qwen2.5-coder:14b ./server.exe
```

For a larger quality-focused model, opt in explicitly:

```bash
ollama pull gemma4:26b
CHAT_MODEL=gemma4:26b ./server.exe
```

## GitHub Actions

This repo includes `.github/workflows/ci.yml` for personal GitHub repositories.

- Pushes and pull requests run `go build`, `go test`, and `go vet` with Go module dependencies.
- A Windows GitHub-hosted runner also compiles the Wails desktop target.
- Semver tags such as `v1.2.0` run GoReleaser using `.goreleaser.github.yaml`.
- Tagged releases also build the Windows NSIS installer from the desktop binary and upload it to the GitHub Release.
- Release builds bake the GitHub repository into the binary so the in-app updater can check GitHub Releases.
- Jobs use standard GitHub-hosted runners, `ubuntu-latest` and `windows-latest`, which are free and unlimited for public repositories.

## Windows Installer

The Windows installer is built from `packaging/windows/jarvis.nsi`.

- Installs per-user to `%LOCALAPPDATA%\Programs\Jarvis`.
- Does not require UAC.
- Manual installs copy and run `bootstrap-ollama.ps1`.
- The bootstrap script checks for Ollama, installs it if missing, starts the local API, and pulls `nomic-embed-text`, `gemma4:e2b`, `gemma4:e4b`, and `llava`.
- Silent installs, including in-app auto-updates, skip the Ollama bootstrap and restart Jarvis after install.

## Desktop App

Jarvis includes a Wails desktop target in `cmd/desktop`. It is the primary installed app and runs the same Go runtime as the browser server.

Use `wails build` for a full Wails production build, or use `go build -tags "desktop,production" ./cmd/desktop` for a quick local compile. Running `go build ./cmd/desktop` without those tags creates a binary that shows Wails' build-tags error dialog.

Desktop mode starts a hidden loopback API server on `127.0.0.1` and injects that API base into the frontend. This keeps the browser and desktop UI shared while preserving token-by-token streaming in the desktop app.
