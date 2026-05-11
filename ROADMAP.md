# Jarvis Roadmap

This document tracks the remaining planned work for Jarvis.

## Product Direction

Jarvis should grow from a private document chatbot into an open-source, web-based local coding agent. The goal is similar to OpenAI Codex, but with local models, local files, and user-controlled execution by default.

The key product bets:

- **Gemma-native and local-first**: build Jarvis around Gemma 4 as the primary model family. Use local inference first, with Ollama kept as a transition/fallback path while llama.cpp support lands.
- **Workspace, not only chat**: keep the browser and desktop UI shared, but evolve Jarvis into a workbench with files, tasks, workspace maps, diffs, terminals, tests, and review output.
- **Trust through visibility**: every agent action should be visible, interruptible, and reversible. Show plans, file reads, commands, patches, test output, retrieval traces, and final diffs.
- **Small models with good context**: use retrieval, workspace maps, symbols, BM25, reranking, and compact task state so local models can perform well without huge memory needs.
- **Human-in-the-loop by default**: proposed edits are reviewed before apply. Commands require allowlists or approval until the user changes policy.

## Research Notes

Recent coding-agent tools point toward a few patterns worth copying:

- OpenAI Codex focuses on delegated engineering tasks, parallel agent runs, cloud sandboxes, code review, pull requests, skills, and automations.
- OpenHands shows that an open, model-agnostic coding agent needs a clear task loop, sandboxed runtime, tool calling, GitHub integration, and reusable skills.
- Continue.dev is strong at IDE-style roles: chat, edit, apply, autocomplete, embeddings, and reranking. Jarvis should adopt the role separation even if it stays web-based.
- Aider is a useful reference for git-aware editing, workspace maps, focused multi-file patches, automatic test runs, and commit-message generation.
- Localforge is a good UI reference for local tasks, visual diffs, model switching, task tracking, and expert/review modes.
- Google's official Gemma 4 31B Hugging Face card is the source of truth for model capabilities: image-text-to-text, Apache 2.0 license, text generation output, text/image input across the family, audio on E2B/E4B, 128K/256K context tiers, configurable thinking, system role support, native function calling, coding, multilingual support, and agentic workflows. Reference: https://huggingface.co/google/gemma-4-31B
- Unsloth's Gemma 4 local guide is a good runtime reference: Gemma 4 GGUFs, llama.cpp execution, `temperature=1.0`, `top_p=0.95`, `top_k=64`, explicit `<|think|>` thinking control, multimodal settings, and the rule to keep only final visible answers in multi-turn history. Reference: https://unsloth.ai/docs/models/gemma-4
- Gemma 4 should be treated as a multimodal understanding model family, not a native media generation stack. E2B and E4B support text, image, and audio input. 26B-A4B and 31B support text and image input. Image generation, audio generation, and video generation should be separate optional model integrations if Jarvis adds creator features later.

## Gemma 4 Runtime Direction

Jarvis should become Gemma-native over time instead of being tightly coupled to Ollama.

| Workstream | What it adds |
|---|---|
| **Gemma 4 model ladder** | Standardize on Gemma 4 E2B for very small machines, E4B as the default fast local model, 26B-A4B for the best speed/quality workstation profile, and 31B for maximum local quality. |
| **llama.cpp runtime** | Add a direct llama.cpp backend for GGUF models, model download/cache management, streaming, cancellation, context sizing, and platform-specific acceleration. Keep Ollama as fallback during migration. |
| **Gemma chat template** | Implement Gemma 4 turn formatting, end-of-sentence handling with `<turn|>`, recommended sampling defaults, and history cleanup so thought blocks are never fed back into later turns. |
| **Thinking profiles** | Add role-aware system prompts: fast chat without thinking, research/reasoning with `<|think|>`, coding-agent planning with thinking, and final-answer cleanup that hides internal reasoning from the UI. |
| **Gemma multimodal understanding** | Use E2B/E4B for text, image, audio, and short video understanding on small machines. Use 26B-A4B or 31B for stronger text/image reasoning when hardware allows. Keep generation features separate unless a dedicated image/audio model is added. |
| **Gemma prompt library** | Keep Gemma prompt profiles in Go code, not Python scripts: direct chat, research, query generation, coding-agent planning, command output analysis, patch review, OCR/document understanding, multimodal comparison, ASR, and speech translation. |
| **Agentic Gemma loop** | Tune project tools, command approvals, patch generation, review mode, and test loops specifically for Gemma 4 tool-use behavior. |

## Phase 3 - Power Features

| Feature | What it adds |
|---|---|
| **Multiple named workspaces** | Save and switch between named document sets. Each workspace should have its own vector store, documents, and chat history. |
| **Vector compression research** | Evaluate TurboQuant, QJL, and PolarQuant ideas for compressing Jarvis embeddings or adding an approximate search tier. Reference: https://research.google/blog/turboquant-redefining-ai-efficiency-with-extreme-compression/ |
| **Workspace map depth** | Add richer document metadata, media dimensions, Office document summaries, package boundaries, tests, and better file grouping. |
| **Context picker** | Let users pin files, folders, symbols, diffs, terminal output, URLs, and documents into the next task. Show context budget impact before sending. |
| **Code-aware retrieval** | Add code-specific chunking, symbol metadata, exact identifier search, dependency-aware boosting, and optional reranking. |
| **Model role profiles** | Split model config by role: chat, edit, apply, autocomplete, embedding, reranker, and vision. |
| **Gemma 4 runtime research** | Define the Gemma-native model ladder, llama.cpp backend plan, thinking profiles, and migration away from mandatory Ollama. |
| **Gemma capability matrix** | In progress. Track text generation, text/image/audio/video input, thinking, function calling, coding, multilingual support, system prompt support, context length, memory guidance, and explicit non-support for native image/audio generation in Go so the UI and installer do not overpromise. |
| **Frontend modularization** | In progress. Chat behavior is split into focused message modules. Next split the large HTML template into embedded partials and organize CSS by component. |

## Recently Completed

These features are already implemented and should not be treated as active roadmap work:

- Wails desktop-first app path with browser server kept for development and fallback
- GitHub Releases updater path and Windows installer packaging
- Windows Ollama bootstrap during manual install
- Workbench activity and workspace map
- Workspace map support for regular folders, Office documents, PDFs, images, spreadsheets, presentations, data, text, config, and code files
- Reply-to-message context and focused `@file` / `#file` retrieval
- Pasted image previews and image indexing through the normal document pipeline
- Queued chat follow-ups with edit, reorder, and remove controls
- Response copy, rating, fork, and working-time controls
- Local git change summary and expandable diff review for the active project
- Read-only project tools for active-project file search, preview, symbols, imports, and summaries
- Approved command runner for allowlisted checks with captured output and task traces
- Watched folder background re-indexing
- Safe clear-index behavior that preserves external source files
- Neutral Codex-style UI theme and updated Help modal
- Gemma 4 capability metadata, prompt guidance, sampling defaults, and thinking-output cleanup
- Project-scoped folder indexes, project-preferred RAG retrieval, and project-tagged chats
- Project-scoped command policy and command history for approved local commands
- Project-scoped edit history and patch-state storage
- First read-only model tool pass that gathers project file summaries before normal chat answers

## Release Planning

The next public tag can stay `v2.1.0`, assuming the last public release was `v2.0.1` and no breaking change is introduced before tagging.

This should be a minor release because it adds backward-compatible product features without requiring users to reconfigure Jarvis.

## Phase 4 - Local Coding Agent

| Feature | What it adds |
|---|---|
| **Agent tool loop** | Let the model call tools mid-task: search files, read files, inspect symbols, list directories, summarize files, fetch URLs, and ask for approval. |
| **Structured model tool calls** | Move from the current deterministic read-only pre-pass to model-requested tool calls with explicit tool names, arguments, traces, and approval prompts. |
| **Safe local command runner in model loop** | Let the agent request allowlisted commands, ask for approval when needed, run with timeouts/output caps, and summarize results. |
| **Patch generation and apply flow** | Generate unified diffs, preview changes in the UI, apply approved patches, and support discard per file. |
| **Test and fix loop** | Let the agent run tests, parse failures, update the patch, and repeat within a bounded iteration count. |
| **Git worktree and branch support** | Create isolated worktrees or branches per task so multiple agents can work without overwriting each other. |
| **Code review mode** | Review a local diff or branch and produce findings with severity, file, line, risk, and test gaps. |
| **GitHub issue and pull request flow** | Pull issue context from GitHub, create a task from it, push a branch, and draft a pull request with summary and tests. |
| **Skills and project rules** | Support reusable local skills and repo instructions through `AGENTS.md` and optional `skills/*/SKILL.md` files. |
| **Gemma-tuned agent prompts** | Add role-specific Gemma 4 prompts for research, planning, coding, patch review, and command output analysis, with thinking enabled only where it helps. |
| **Direct llama.cpp adapter** | Add a Go-managed llama.cpp runtime adapter before removing Ollama: discover/download GGUF models, launch `llama-server` or a bundled runtime safely, stream responses, cancel requests, and map Gemma role profiles to llama.cpp options. Keep the code pure Go by treating llama.cpp as an external runtime process unless the project deliberately accepts CGo later. |

## Agent Workflow Architecture

The Codex-like workflow is possible, but it should land in careful layers:

1. **Structured model tool calls:** let the model request read-only tools, fetch context, and ask for approval through explicit tool calls.
2. **Command tools:** connect the existing allowlisted runner to the model loop with working directory controls, timeouts, output capture, and approval gates.
3. **Patch tools:** propose file edits as diffs, show review UI, apply only approved patches, and record edit history.
4. **Review and test loop:** run tests, parse failures, update patches, and produce a final review summary.

Scheduled folder re-indexing helps this directly because it keeps the active project's local context current while the agent works.
