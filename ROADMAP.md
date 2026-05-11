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
- Unsloth's Gemma 4 local guide is a good runtime reference: Gemma 4 GGUFs, llama.cpp execution, `temperature=1.0`, `top_p=0.95`, `top_k=64`, explicit `<|think|>` thinking control, multimodal settings, and the rule to keep only final visible answers in multi-turn history. Reference: https://unsloth.ai/docs/models/gemma-4

## Gemma 4 Runtime Direction

Jarvis should become Gemma-native over time instead of being tightly coupled to Ollama.

| Workstream | What it adds |
|---|---|
| **Gemma 4 model ladder** | Standardize on Gemma 4 E2B for very small machines, E4B as the default fast local model, 26B-A4B for the best speed/quality workstation profile, and 31B for maximum local quality. |
| **llama.cpp runtime** | Add a direct llama.cpp backend for GGUF models, model download/cache management, streaming, cancellation, context sizing, and platform-specific acceleration. Keep Ollama as fallback during migration. |
| **Gemma chat template** | Implement Gemma 4 turn formatting, end-of-sentence handling with `<turn|>`, recommended sampling defaults, and history cleanup so thought blocks are never fed back into later turns. |
| **Thinking profiles** | Add role-aware system prompts: fast chat without thinking, research/reasoning with `<|think|>`, coding-agent planning with thinking, and final-answer cleanup that hides internal reasoning from the UI. |
| **Gemma multimodal path** | Use E2B/E4B for text, image, and audio features on small machines. Use 26B-A4B or 31B for stronger text/image reasoning when hardware allows. |
| **Agentic Gemma loop** | Tune project tools, command approvals, patch generation, review mode, and test loops specifically for Gemma 4 tool-use behavior. |

## Phase 3 - Power Features

| Feature | What it adds |
|---|---|
| **Workbench activity panel** | Done. Shows task traces, retrieval events, selected model, message count, edit count, and a polished activity timeline. |
| **Workspace map** | Done. Folder ingest maps regular documents, PDFs, images, spreadsheets, presentations, data files, source files, and code symbols. |
| **Reply and focused file context** | Done. Users can reply to a specific message and use `@file` or `#file` mentions to focus retrieval on indexed files. |
| **Windows installer bootstrap** | Done for Windows. Manual NSIS installs check/install Ollama and pull Jarvis models. Silent auto-updates skip bootstrap. |
| **Scheduled folder re-indexing** | Done. Folder ingest saves watched folders and refreshes new or modified files in the background while keeping workspace maps current. |
| **Codex-style chat workspace** | Done. Neutral desktop theme, queued follow-ups, editable/reorderable queue, response copy/rating/fork controls, response timing, and expanded in-app Help. |
| **Natural live context** | Done. Normal chat quietly uses current local time, locale, timezone, pasted URLs, and live web context when available. Research mode remains the transparent source-heavy path. |
| **Project foundation** | Done. Indexed folders become persisted projects with an active project, workspace map, watched re-indexing, and a reserved per-project vector-store path for the agent workflow. |
| **Local diff review** | Done. The Workbench shows changed-file totals, additions, deletions, statuses, and expandable text patches for the active git project. |
| **Multiple named workspaces** | Save and switch between named document sets. Each workspace should have its own vector store, documents, and chat history. |
| **Vector compression research** | Evaluate TurboQuant, QJL, and PolarQuant ideas for compressing Jarvis embeddings or adding an approximate search tier. Reference: https://research.google/blog/turboquant-redefining-ai-efficiency-with-extreme-compression/ |
| **Workspace map depth** | Add richer document metadata, media dimensions, Office document summaries, package boundaries, tests, and better file grouping. |
| **Context picker** | Let users pin files, folders, symbols, diffs, terminal output, URLs, and documents into the next task. Show context budget impact before sending. |
| **Code-aware retrieval** | Add code-specific chunking, symbol metadata, exact identifier search, dependency-aware boosting, and optional reranking. |
| **Model role profiles** | Split model config by role: chat, edit, apply, autocomplete, embedding, reranker, and vision. |
| **Gemma 4 runtime research** | Define the Gemma-native model ladder, llama.cpp backend plan, thinking profiles, and migration away from mandatory Ollama. |
| **Frontend modularization** | In progress. Chat behavior is split into focused message modules. Next split the large HTML template into embedded partials and organize CSS by component. |

## v2.1.0 Release Scope

Target tag: `v2.1.0`, assuming the last public release was `v2.0.1`.

This should be a minor release because it adds backward-compatible product features without requiring users to reconfigure Jarvis.

Included:

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

## Phase 4 - Local Coding Agent

| Feature | What it adds |
|---|---|
| **Project-scoped indexes** | Move from one global vector store to project/document-set scoped stores. Keep each project isolated for retrieval, chat history, workspace map, command policy, and future edits. |
| **Agent tool loop** | Let the model call tools mid-task: search files, read files, inspect symbols, list directories, summarize files, fetch URLs, and ask for approval. |
| **Read-only project tools** | In progress. Search active-project files, read safe text previews, summarize symbols/imports, and record tool traces. Next wire these tools into the model loop. |
| **Safe local command runner** | In progress. Runs only allowlisted, approved commands in the active project with timeouts, output caps, exit code, and task traces. |
| **Patch generation and apply flow** | Generate unified diffs, preview changes in the UI, apply approved patches, and support discard per file. |
| **Test and fix loop** | Let the agent run tests, parse failures, update the patch, and repeat within a bounded iteration count. |
| **Git worktree and branch support** | Create isolated worktrees or branches per task so multiple agents can work without overwriting each other. |
| **Code review mode** | Review a local diff or branch and produce findings with severity, file, line, risk, and test gaps. |
| **GitHub issue and pull request flow** | Pull issue context from GitHub, create a task from it, push a branch, and draft a pull request with summary and tests. |
| **Skills and project rules** | Support reusable local skills and repo instructions through `AGENTS.md` and optional `skills/*/SKILL.md` files. |
| **Gemma-tuned agent prompts** | Add role-specific Gemma 4 prompts for research, planning, coding, patch review, and command output analysis, with thinking enabled only where it helps. |

## Agent Workflow Architecture

The Codex-like workflow is possible, but it should land in careful layers:

1. **Projects:** a folder becomes a persisted project with a root path, active state, workspace map, watched re-indexing, and a reserved vector-store location.
2. **Project-scoped storage:** each project gets its own vector store, chat state, workspace map, command policy, and edit history. This prevents one project from polluting another project's retrieval.
3. **Read-only tools:** list files, search files, read files, inspect symbols, summarize files, and fetch project context. Search, read, and summarize endpoints now exist for active-project files.
4. **Command tools:** run approved commands with working directory controls, timeouts, output capture, and allowlists. The first allowlisted command runner is available for git status/diff and Go test checks.
5. **Patch tools:** propose file edits as diffs, show review UI, apply only approved patches, and record edit history. The first read-only diff review surface is now available in the Workbench.
6. **Review and test loop:** run tests, parse failures, update patches, and produce a final review summary.

Scheduled folder re-indexing helps this directly because it keeps the active project's local context current while the agent works.
