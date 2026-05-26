window.jarvisWorkbench = {
        async loadTaskState() {
            try {
                const resp = await fetch(this.apiURL('/api/v1/task'));
                if (resp.ok) {
                    this.taskState = await resp.json();
                }
            } catch {}
        },

        async loadRepoMap() {
            try {
                const resp = await fetch(this.apiURL('/api/v1/repo-map'));
                if (resp.ok) {
                    this.repoMap = await resp.json();
                } else if (resp.status === 404) {
                    this.repoMap = null;
                }
            } catch {}
        },

        async loadProjects() {
            try {
                const resp = await fetch(this.apiURL('/api/v1/projects'));
                if (resp.ok) {
                    this.projectState = await resp.json();
                    await this.loadDocuments();
                    await this.loadProjectActivity();
                    await this.refreshChatList();
                }
            } catch {}
        },

        async loadProjectActivity() {
            try {
                const resp = await fetch(this.apiURL('/api/v1/projects/activity'));
                if (resp.ok) {
                    this.projectActivity = await resp.json();
                } else if (resp.status === 404) {
                    this.projectActivity = null;
                }
            } catch {
                this.projectActivity = null;
            }
        },

        async loadDiffSummary() {
            try {
                const resp = await fetch(this.apiURL('/api/v1/diff'));
                if (resp.ok) {
                    this.diffSummary = await resp.json();
                } else {
                    this.diffSummary = null;
                }
            } catch {
                this.diffSummary = null;
            }
        },

        async searchProjectTools() {
            this.toolLoading = true;
            try {
                const params = new URLSearchParams();
                if ((this.toolSearch || '').trim()) {
                    params.set('query', this.toolSearch.trim());
                }
                params.set('limit', '40');
                const resp = await fetch(this.apiURL(`/api/v1/tools/files?${params.toString()}`));
                if (!resp.ok) throw new Error('Search failed');
                const data = await resp.json();
                this.toolResults = data.files || [];
                if (this.toolResults.length === 0) {
                    this.toolPreview = null;
                }
            } catch {
                this.toolResults = [];
                this.showToast('Project file search failed', 'error');
            } finally {
                this.toolLoading = false;
            }
        },

        async previewProjectFile(path) {
            this.toolLoading = true;
            try {
                const resp = await fetch(this.apiURL('/api/v1/tools/summarize-file'), {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path })
                });
                if (!resp.ok) throw new Error('Preview failed');
                this.toolPreview = await resp.json();
            } catch {
                this.showToast('File preview failed', 'error');
            } finally {
                this.toolLoading = false;
            }
        },

        async runProjectCommand() {
            const parsed = this.parseCommandPreset(this.commandPreset);
            if (!parsed) {
                this.showToast('Choose an approved command first', 'error');
                return;
            }
            this.commandRunning = true;
            try {
                const resp = await fetch(this.apiURL('/api/v1/tools/run-command'), {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        command: parsed.command,
                        args: parsed.args,
                        approved: true,
                        timeout_seconds: 60
                    })
                });
                if (!resp.ok) throw new Error('Command failed');
                this.commandResult = await resp.json();
                await this.loadTaskState();
            } catch {
                this.showToast('Command runner failed', 'error');
            } finally {
                this.commandRunning = false;
            }
        },

        parseCommandPreset(value) {
            const presets = {
                'git status --short': { command: 'git', args: ['status', '--short'] },
                'git diff --stat': { command: 'git', args: ['diff', '--stat'] },
                'git diff --name-only': { command: 'git', args: ['diff', '--name-only'] },
                'go test ./...': { command: 'go', args: ['test', './...'] }
            };
            return presets[value] || null;
        },

        async openWorkbenchPanel() {
            await Promise.all([
                this.loadTaskState(),
                this.loadRepoMap(),
                this.loadProjects(),
                this.loadProjectActivity(),
                this.loadDiffSummary()
            ]);
            this.showWorkbenchPanel = true;
        },

        async clearTaskState() {
            if (!confirm('Clear local workbench activity? Chat history and indexed files stay intact.')) return;
            try {
                const resp = await fetch(this.apiURL('/api/v1/task'), { method: 'DELETE' });
                if (!resp.ok) throw new Error('Clear failed');
                await this.loadTaskState();
                this.showToast('Workbench activity cleared');
            } catch {
                this.showToast('Failed to clear workbench activity', 'error');
            }
        },

        workbenchSummary() {
            const task = this.taskState || {};
            return {
                traces: (task.traces || []).length,
                messages: (task.messages || []).length,
                edits: (task.edit_history || []).length,
                model: task.selected_model || this.systemInfo?.chat_model || ''
            };
        },

        projectSummary() {
            const state = this.projectState || {};
            const projects = state.projects || [];
            const active = projects.find(project => project.active) || null;
            const activeDocumentCount = active ? (this.documents || []).length : 0;
            const activity = this.projectActivity || {};
            return {
                count: projects.length,
                active,
                vectorStoreDir: active?.vector_store_dir || '',
                activeDocumentCount,
                commandCount: (activity.command_history || []).length,
                approvedCommandCount: (activity.command_policy?.approved_commands || []).length,
                editCount: (activity.edit_history || []).length,
                patchCount: (activity.patch_history || []).length,
                retrievalScope: activeDocumentCount > 0 ? 'Project index active' : 'Global index fallback'
            };
        },

        repoSummary() {
            const repo = this.repoMap || {};
            return {
                root: repo.root || '',
                files: repo.file_count || (repo.files || []).length || 0,
                symbols: repo.symbol_count || (repo.symbols || []).length || 0,
                tests: repo.test_count || 0,
                packages: (repo.packages || []).length,
                groups: (repo.groups || []).length,
                updated: repo.updated_at || ''
            };
        },

        diffSummaryStats() {
            const summary = this.diffSummary || {};
            return {
                root: summary.root || '',
                files: summary.file_count || (summary.files || []).length || 0,
                additions: summary.additions || 0,
                deletions: summary.deletions || 0
            };
        },

        toggleDiff(path) {
            this.expandedDiffs = {
                ...this.expandedDiffs,
                [path]: !this.expandedDiffs[path]
            };
        },

        isDiffExpanded(path) {
            return Boolean(this.expandedDiffs?.[path]);
        },

        toggleToolTrace(trace) {
            const key = this.toolTraceKey(trace);
            this.expandedToolTraces = {
                ...this.expandedToolTraces,
                [key]: !this.expandedToolTraces[key]
            };
        },

        isToolTraceExpanded(trace) {
            return Boolean(this.expandedToolTraces?.[this.toolTraceKey(trace)]);
        },

        toolTraceKey(trace) {
            return [
                trace?.type || 'tool',
                trace?.summary || '',
                trace?.created_at || ''
            ].join('|');
        },

        agentToolTraces(limit = 12) {
            const task = this.taskState || {};
            return (task.traces || [])
                .filter(trace => this.isAgentToolTrace(trace))
                .sort((a, b) => new Date(b.created_at || 0) - new Date(a.created_at || 0))
                .slice(0, limit);
        },

        isAgentToolTrace(trace) {
            const type = (trace?.type || '').toLowerCase();
            return type === 'tool_plan' ||
                type === 'tool_call' ||
                type === 'tool_read' ||
                type === 'tool_command';
        },

        toolTraceLabel(trace) {
            const type = (trace?.type || '').toLowerCase();
            const labels = {
                tool_plan: 'Plan',
                tool_call: 'Tool',
                tool_read: 'Read',
                tool_command: 'Command'
            };
            return labels[type] || 'Tool';
        },

        toolTraceDetailEntries(trace) {
            return Object.entries(trace?.metadata || {})
                .filter(([key, value]) => key && value)
                .map(([key, value]) => ({ key, value }));
        },

        diffLines(file, limit = 500) {
            const patch = file?.patch || 'No text patch available.';
            return patch.split('\n').slice(0, limit);
        },

        diffLineClass(line) {
            if (line.startsWith('+') && !line.startsWith('+++')) return 'is-add';
            if (line.startsWith('-') && !line.startsWith('---')) return 'is-delete';
            if (line.startsWith('@@')) return 'is-hunk';
            if (line.startsWith('diff --git')) return 'is-header';
            return '';
        },

        diffStatusLabel(file) {
            if (file?.binary) return `${file.status || 'changed'} binary`;
            return file?.status || 'changed';
        },

        toolPreviewLines(limit = 80) {
            const excerpt = this.toolPreview?.excerpt || this.toolPreview?.description || '';
            return excerpt.split('\n').slice(0, limit);
        },

        commandOutputLines(limit = 180) {
            const result = this.commandResult || {};
            const output = [result.stdout || '', result.stderr || '']
                .filter(Boolean)
                .join('\n')
                .trim();
            return (output || 'No output').split('\n').slice(0, limit);
        },

        workspaceTypeBreakdown(limit = 8) {
            const counts = new Map();
            for (const file of this.repoMap?.files || []) {
                const type = file.kind || file.language || 'file';
                counts.set(type, (counts.get(type) || 0) + 1);
            }
            return Array.from(counts.entries())
                .map(([type, count]) => ({ type, count }))
                .sort((a, b) => b.count - a.count || a.type.localeCompare(b.type))
                .slice(0, limit);
        },

        topWorkspaceGroups(limit = 5) {
            return [...(this.repoMap?.groups || [])]
                .sort((a, b) => (b.file_count || 0) - (a.file_count || 0) || (a.path || '').localeCompare(b.path || ''))
                .slice(0, limit);
        },

        topWorkspacePackages(limit = 5) {
            return [...(this.repoMap?.packages || [])]
                .sort((a, b) => (b.file_count || 0) - (a.file_count || 0) || (a.path || '').localeCompare(b.path || ''))
                .slice(0, limit);
        },

        filteredWorkspaceFiles(limit = 80) {
            const files = this.repoMap?.files || [];
            const query = (this.repoSearch || '').trim().toLowerCase();
            return files
                .filter(file => {
                    if (!query) return true;
                    return (
                        (file.path || '').toLowerCase().includes(query) ||
                        (file.kind || '').toLowerCase().includes(query) ||
                        (file.language || '').toLowerCase().includes(query)
                    );
                })
                .sort((a, b) => (a.path || '').localeCompare(b.path || ''))
                .slice(0, limit);
        },

        filteredSymbols(limit = 50) {
            const symbols = this.repoMap?.symbols || [];
            const query = (this.repoSearch || '').trim().toLowerCase();
            return symbols
                .filter(symbol => {
                    if (!query) return true;
                    return (
                        (symbol.name || '').toLowerCase().includes(query) ||
                        (symbol.kind || '').toLowerCase().includes(query) ||
                        (symbol.file_path || '').toLowerCase().includes(query) ||
                        (symbol.language || '').toLowerCase().includes(query)
                    );
                })
                .sort((a, b) => {
                    const fileCompare = (a.file_path || '').localeCompare(b.file_path || '');
                    if (fileCompare !== 0) return fileCompare;
                    return (a.line || 0) - (b.line || 0);
                })
                .slice(0, limit);
        },

        recentWorkbenchEvents(limit = 30) {
            const task = this.taskState || {};
            const events = [];

            for (const trace of task.traces || []) {
                events.push({
                    kind: 'trace',
                    type: trace.type || 'trace',
                    title: trace.summary || 'Trace event',
                    detail: this.metadataSummary(trace.metadata),
                    created_at: trace.created_at
                });
            }
            for (const edit of task.edit_history || []) {
                events.push({
                    kind: 'edit',
                    type: edit.action || 'edit',
                    title: edit.path || 'Edit',
                    detail: edit.summary || '',
                    created_at: edit.created_at
                });
            }

            return events
                .sort((a, b) => new Date(b.created_at || 0) - new Date(a.created_at || 0))
                .slice(0, limit);
        },

        metadataSummary(metadata) {
            if (!metadata) return '';
            return Object.entries(metadata)
                .filter(([key, value]) => key && value)
                .map(([key, value]) => `${key}: ${value}`)
                .join(' , ');
        },

        eventTime(value) {
            if (!value) return '';
            try {
                return new Date(value).toLocaleString([], {
                    month: 'short',
                    day: 'numeric',
                    hour: '2-digit',
                    minute: '2-digit'
                });
            } catch {
                return '';
            }
        },

        eventTone(type) {
            const normalized = (type || '').toLowerCase();
            if (normalized.includes('error') || normalized.includes('failed')) return 'is-error';
            if (normalized.includes('edit')) return 'is-edit';
            if (normalized.includes('research')) return 'is-research';
            if (normalized.includes('retrieval') || normalized.includes('repo')) return 'is-context';
            return '';
        },
};
