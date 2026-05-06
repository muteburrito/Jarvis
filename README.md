# Jarvis - Chat with your Documents

A local, privacy-first document chatbot built with Go and Alpine.js. Upload PDFs, DOCX, XLSX, code files, or point it at an entire project folder. It indexes everything locally and lets you ask questions using a local LLM through Ollama. It can also search the web and research topics for you.

No data leaves your machine. No API keys needed. Single binary, runs anywhere.

## Features

- **Chat with documents:** upload files or index entire folders, then ask questions with source citations
- **General chat:** works as a regular assistant even without documents loaded
- **Streaming responses:** token-by-token SSE streaming with markdown rendering
- **Recursive folder indexing:** point it at a C#, Go, Python (or any) project and it walks all subdirectories, skipping build output like `bin/`, `obj/`, `node_modules/`
- **Wide file support:** PDF, DOCX, XLSX, PPTX, images, known source files, and unknown text-like files. Binary files are rejected
- **Hybrid retrieval:** combines vector similarity with BM25 keyword scoring for better exact matches on code symbols, error codes, and config keys
- **Codebase memory:** folder ingest builds a lightweight repo map, symbol index, task state, and trace log for Codex-style coding sessions
- **Image support:** upload standalone images (PNG, JPG, etc.) or PDFs with embedded images. A vision model describes each image so it becomes searchable and queryable
- **Deep research mode:** toggle research mode and DocChat searches the web via DuckDuckGo, fetches the top articles, indexes them, and answers with citations and links. No API key needed
- **URL fetching:** paste a website link directly in chat. DocChat auto-detects URLs in normal chat messages, fetches the page, and indexes it. URLs inside the code snippet box are treated as code and are not fetched
- **Regional awareness:** automatically detects your locale and timezone from the browser. Answers use your local currency, date formats, and regionally relevant context
- **Document management:** upload, list, delete individual docs, or clear everything at once
- **Server-side chat history:** chats are saved locally under the data directory and shown in the sidebar
- **Dark mode UI:** clean, responsive interface built with Tailwind CSS and Alpine.js
- **Chat-bar attachments** with file and folder picker buttons plus progress tracking
- **Code snippet mode** with automatic language detection and syntax-highlighted preview
- **Self-contained binary:** the web UI is embedded into the exe via go:embed. No external files needed. Pure Go, no CGo, builds on Windows/Linux/Mac

## Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Ollama](https://ollama.com/) running locally

Pull the required models:

```bash
ollama pull gemma4:e2b
ollama pull nomic-embed-text
ollama pull llava              # optional, enables image support
```

Jarvis picks the chat model automatically when `CHAT_MODEL` is not set. Very low-end machines use `gemma4:e2b`; everyone else uses `gemma4:e4b`. The larger `26b` model is not selected automatically because Jarvis is local and RAG-first, and speed matters more for daily use. If Ollama is running but the required chat or embedding model is missing, Jarvis downloads it automatically on startup. Manual pulls are still useful for preparing a machine ahead of time.

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

## Release Versioning

Jarvis uses semantic versioning for release tags: `vMAJOR.MINOR.PATCH`, for example `v1.4.2`. The GitHub updater only runs on proper semver builds, so release tags should always use this format.

- **Major**: increment for breaking changes. Examples: incompatible config changes, vector store format changes without migration, removed APIs, changed install locations, or behavior that requires users to manually reconfigure Jarvis.
- **Minor**: increment for backward-compatible features. Examples: new document loaders, new UI features, new API endpoints, new installer targets, new model options, or safe storage migrations.
- **Patch**: increment for backward-compatible fixes. Examples: bug fixes, security fixes, log noise cleanup, small UI polish, CI fixes, packaging fixes, and documentation corrections.

When creating a release, build with the same version baked into the binary:

```bash
go build -ldflags="-X main.Version=v1.4.2" -o jarvis.exe ./cmd/server/
```

## Configuration

All settings are configurable through environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `OLLAMA_URL` | `http://localhost:11434` | Ollama API endpoint |
| `OLLAMA_KEEP_ALIVE` | `30s` | How long Ollama keeps models loaded after a request. Use `0s` for lowest idle memory |
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
cmd/server/main.go            Entry point, wires everything together
internal/
  config/config.go             Environment-based configuration
  ollama/client.go             Ollama API wrapper (chat, embeddings, vision)
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
    repo.go                    Repo map and symbol scanning
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
  static/css/app.css           Custom styles
  embed.go                     Embeds web/ into the binary
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
| `GET /api/v1/repo-map` | Current lightweight repo map with files, imports, and symbols |
| `GET /api/v1/task` | Current persisted task state, messages, traces, and edit history |
| `POST /api/v1/task/traces` | Append a tool trace event to the current task |
| `POST /api/v1/task/edits` | Append an edit-history entry to the current task |
| `DELETE /api/v1/task` | Clear current task state |

## How It Works

1. **Upload or index:** files are split into overlapping text chunks. Images (standalone or extracted from PDFs) are described by a vision model, and those descriptions become searchable text. URLs pasted in the main chat box are auto-detected and fetched.
2. **Embed:** each chunk is converted to a vector using `nomic-embed-text` via Ollama
3. **Store:** vectors are kept in memory and persisted to disk in gob format
4. **Query:** your question is embedded, the most similar chunks are retrieved, and they are passed as context to the LLM. Your locale and timezone are included so answers use local conventions.
5. **Stream:** the LLM response streams back token-by-token via Server-Sent Events

### Research mode

Toggle the magnifying glass button next to the chat input. When active:

1. The LLM generates 2-3 focused search queries from your question
2. Each query is searched on DuckDuckGo (no API key needed)
3. The top results are fetched and indexed through the same pipeline as uploaded files
4. The LLM then answers using the fetched content, with citations and source links
5. Fetched articles stay in your vector store, so follow-up questions reuse them without re-fetching

Research steps stream as a compact progress timeline while the answer is being prepared.

## Hardware Requirements

Ollama runs the models on your hardware. DocChat itself (the Go server) uses very little memory, typically under 100 MB even with thousands of indexed chunks. The models are what need the horsepower.

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
- Semver tags such as `v1.2.0` run GoReleaser using `.goreleaser.github.yaml`.
- Tagged releases also build the Windows NSIS installer and upload it to the GitHub Release.
- Release builds bake the GitHub repository into the binary so the in-app updater can check GitHub Releases.
- Jobs use standard GitHub-hosted runners, `ubuntu-latest` and `windows-latest`, which are free and unlimited for public repositories.
