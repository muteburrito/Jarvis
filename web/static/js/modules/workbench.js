window.jarvisWorkbench = {
        async loadTaskState() {
            try {
                const resp = await fetch(this.apiURL('/api/v1/task'));
                if (resp.ok) {
                    this.taskState = await resp.json();
                }
            } catch {}
        },

        async openWorkbenchPanel() {
            await this.loadTaskState();
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
