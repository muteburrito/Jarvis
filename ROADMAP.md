# Jarvis Roadmap

This document tracks the remaining planned work for Jarvis.

## Product Direction

Jarvis should grow from a private document chatbot into an open-source, web-based local coding agent. The goal is similar to OpenAI Codex, but with local models, local files, and user-controlled execution by default.

The key product bets:

- **Local-first and open-source**: run with Ollama or other local model servers first. Cloud models can be optional later, but never required.
- **Web workbench, not only chat**: keep the browser UI, but evolve it into a coding workspace with files, tasks, diffs, terminals, tests, and review output.
- **Trust through visibility**: every agent action should be visible, interruptible, and reversible. Show plans, file reads, commands, patches, test output, and final diffs.
- **Small models with good context**: use retrieval, repo maps, symbols, BM25, reranking, and compact task state so local models can perform well without huge memory needs.
- **Human-in-the-loop by default**: proposed edits are reviewed before apply. Commands require allowlists or approval until the user changes policy.

## Research Notes

Recent coding-agent tools point toward a few patterns worth copying:

- OpenAI Codex focuses on delegated engineering tasks, parallel agent runs, cloud sandboxes, code review, pull requests, skills, and automations.
- OpenHands shows that an open, model-agnostic coding agent needs a clear task loop, sandboxed runtime, tool calling, GitHub integration, and reusable skills.
- Continue.dev is strong at IDE-style roles: chat, edit, apply, autocomplete, embeddings, and reranking. Jarvis should adopt the role separation even if it stays web-based.
- Aider is a useful reference for git-aware editing, repo maps, focused multi-file patches, automatic test runs, and commit-message generation.
- Localforge is a good UI reference for local tasks, visual diffs, model switching, task tracking, and expert/review modes.

## Phase 3 - Power Features

| Feature | What it adds |
|---|---|
| **Scheduled folder re-indexing** | Watch folders for file changes and re-index modified or new files in the background. Useful for live wikis, shared drives, and actively developed codebases. |
| **Multiple named workspaces** | Save and switch between named document sets. Each workspace should have its own vector store, documents, and chat history. |
| **Vector compression research** | Evaluate TurboQuant, QJL, and PolarQuant ideas for compressing Jarvis embeddings or adding an approximate search tier. Reference: https://research.google/blog/turboquant-redefining-ai-efficiency-with-extreme-compression/ |
| **Codebase map** | Build a lightweight repo map with files, symbols, imports, package or module boundaries, and tests. |
| **Context picker** | Let users pin files, folders, symbols, diffs, terminal output, URLs, and documents into the next task. Show context budget impact before sending. |
| **Code-aware retrieval** | Add code-specific chunking, symbol metadata, exact identifier search, dependency-aware boosting, and optional reranking. |
| **Model role profiles** | Split model config by role: chat, edit, apply, autocomplete, embedding, reranker, and vision. |
| **Frontend modularization** | Split the large Alpine app and HTML template into smaller maintainable modules or partials while keeping the current no-build setup unless explicitly changed. |

## Phase 4 - Local Coding Agent

| Feature | What it adds |
|---|---|
| **Agent tool loop** | Let the model call tools mid-task: search files, read files, inspect symbols, list directories, summarize files, fetch URLs, and ask for approval. |
| **Safe local command runner** | Add a terminal tool with working directory controls, timeouts, output capture, and command approval policies. |
| **Patch generation and apply flow** | Generate unified diffs, preview changes in the UI, apply approved patches, and support discard per file. |
| **Test and fix loop** | Let the agent run tests, parse failures, update the patch, and repeat within a bounded iteration count. |
| **Git worktree and branch support** | Create isolated worktrees or branches per task so multiple agents can work without overwriting each other. |
| **Code review mode** | Review a local diff or branch and produce findings with severity, file, line, risk, and test gaps. |
| **GitHub issue and pull request flow** | Pull issue context from GitHub, create a task from it, push a branch, and draft a pull request with summary and tests. |
| **Skills and project rules** | Support reusable local skills and repo instructions through `AGENTS.md` and optional `skills/*/SKILL.md` files. |
