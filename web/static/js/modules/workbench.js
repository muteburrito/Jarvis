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
                }
            } catch {}
        },

        async openWorkbenchPanel() {
            await Promise.all([
                this.loadTaskState(),
                this.loadRepoMap(),
                this.loadProjects()
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
            return {
                count: projects.length,
                active,
                vectorStoreDir: active?.vector_store_dir || ''
            };
        },

        repoSummary() {
            const repo = this.repoMap || {};
            return {
                root: repo.root || '',
                files: repo.file_count || (repo.files || []).length || 0,
                symbols: repo.symbol_count || (repo.symbols || []).length || 0,
                updated: repo.updated_at || ''
            };
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
