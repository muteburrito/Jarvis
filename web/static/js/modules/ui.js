window.jarvisUi = {
        renderMarkdown(text) {
            if (!text) return '';
            try {
                const html = marked.parse(text);
                return DOMPurify.sanitize(html);
            } catch {
                return DOMPurify.sanitize(text);
            }
        },

        highlightCode(code, language = '') {
            const text = code || '';
            const normalized = this.normalizeHighlightLanguage(language);
            if (window.hljs) {
                try {
                    if (normalized && hljs.getLanguage(normalized)) {
                        return hljs.highlight(text, { language: normalized, ignoreIllegals: true }).value;
                    }
                    return hljs.highlightAuto(text).value;
                } catch {}
            }
            return this.escapeHtml(text);
        },

        detectedCodeLanguage() {
            return this.detectCodeLanguage(this.codeSnippet);
        },

        codeSnippetLineCount() {
            const text = this.codeSnippet || '';
            if (!text.trim()) return 0;
            return text.split(/\r\n|\r|\n/).length;
        },

        formatBytes(size) {
            const bytes = Number(size) || 0;
            if (bytes <= 0) return '0 B';

            const units = ['B', 'KB', 'MB', 'GB'];
            const unitIndex = Math.min(
                Math.floor(Math.log(bytes) / Math.log(1024)),
                units.length - 1
            );
            const value = bytes / Math.pow(1024, unitIndex);
            const precision = unitIndex === 0 || value >= 10 ? 0 : 1;
            return `${value.toFixed(precision)} ${units[unitIndex]}`;
        },

        formatDuration(milliseconds) {
            const totalSeconds = Math.max(0, Math.floor((Number(milliseconds) || 0) / 1000));
            const hours = Math.floor(totalSeconds / 3600);
            const minutes = Math.floor((totalSeconds % 3600) / 60);
            const seconds = totalSeconds % 60;

            if (hours > 0) {
                return `${hours}h ${minutes}m`;
            }
            if (minutes > 0) {
                return `${minutes}m ${seconds}s`;
            }
            return `${seconds}s`;
        },

        detectCodeLanguage(code) {
            const text = code || '';
            const trimmed = text.trim();
            if (!trimmed) return 'text';

            if (/^\s*package\s+\w+/m.test(text) || /\bfunc\s+\w+\s*\(/.test(text) || /\bfmt\.\w+\(/.test(text)) return 'go';
            if (
                /\busing\s+System\b/.test(text) ||
                /\bnamespace\s+\w+/.test(text) ||
                /\bpublic\s+(class|record|interface)\b/.test(text)
            ) {
                return 'csharp';
            }
            if (/\b(def|class)\s+\w+\s*\(/.test(text) || /\bimport\s+\w+/.test(text) && /:\s*(#.*)?$/m.test(text)) return 'python';
            if (/\b(interface|type)\s+\w+\s*=/.test(text) || /:\s*(string|number|boolean)\b/.test(text)) return 'typescript';
            if (/\b(function|const|let|var)\s+\w+/.test(text) || /=>/.test(text)) return 'javascript';
            if (/^\s*[{[]/.test(trimmed)) {
                try {
                    JSON.parse(trimmed);
                    return 'json';
                } catch {}
            }
            if (/^\s*<([a-z][\w-]*)(\s|>)/i.test(trimmed)) return 'html';
            if (/^\s*[\w.-]+:\s+/m.test(text) && !/[;{}]/.test(text)) return 'yaml';
            if (/\bSELECT\b|\bFROM\b|\bWHERE\b/i.test(text)) return 'sql';
            if (/^\s*#include\s+/.test(text) || /\bint\s+main\s*\(/.test(text)) return 'cpp';
            if (window.hljs) {
                try {
                    const result = hljs.highlightAuto(text);
                    if (result.language) return result.language;
                } catch {}
            }
            return 'text';
        },

        normalizeHighlightLanguage(language) {
            const lang = (language || '').toLowerCase();
            const aliases = {
                cs: 'csharp',
                csharp: 'csharp',
                'c#': 'csharp',
                js: 'javascript',
                ts: 'typescript',
                golang: 'go',
                yml: 'yaml',
                sh: 'bash',
                shell: 'bash',
                ps1: 'powershell',
                py: 'python',
                html: 'xml'
            };
            return aliases[lang] || lang;
        },

        escapeHtml(text) {
            return (text || '')
                .replace(/&/g, '&amp;')
                .replace(/</g, '&lt;')
                .replace(/>/g, '&gt;')
                .replace(/"/g, '&quot;')
                .replace(/'/g, '&#039;');
        },

        async copyCodeBlock(button) {
            const wrapper = button.closest('.code-block-wrapper');
            const code = wrapper?.querySelector('pre code')?.innerText || '';
            if (!code) return;

            try {
                await navigator.clipboard.writeText(code);
                const original = button.textContent;
                button.textContent = 'Copied';
                button.classList.add('is-copied');
                setTimeout(() => {
                    button.textContent = original || 'Copy';
                    button.classList.remove('is-copied');
                }, 1200);
            } catch {
                this.showToast('Could not copy code', 'error');
            }
        },

        autoResize(el) {
            el.style.height = 'auto';
            el.style.height = Math.min(el.scrollHeight, 120) + 'px';
        },

        isNearMessageBottom(threshold = 96) {
            const container = this.$refs.messagesContainer;
            if (!container) return true;
            const distance = container.scrollHeight - container.scrollTop - container.clientHeight;
            return distance <= threshold;
        },

        scrollToBottom(force = false) {
            const shouldScroll = force || this.isNearMessageBottom();
            this.$nextTick(() => {
                const container = this.$refs.messagesContainer;
                if (container && shouldScroll) {
                    container.scrollTop = container.scrollHeight;
                }
            });
        },

        showToast(message, type = 'success') {
            this.toast = { show: true, message, type };
            setTimeout(() => { this.toast.show = false; }, 3000);
        },

        exportConversation() {
            if (this.messages.length === 0) {
                this.showToast('No conversation to export', 'error');
                return;
            }

            const appName = this.appConfig.app_name || 'Jarvis';
            const lines = [`# ${appName} Conversation`, '', `Exported: ${new Date().toLocaleString()}`, ''];
            for (const message of this.messages) {
                const speaker = message.role === 'user' ? 'User' : appName;
                lines.push(`## ${speaker}`, '', message.content || '', '');
                if (message.sources && message.sources.length > 0) {
                    lines.push('### Sources', '');
                    for (const source of message.sources) {
                        lines.push(`- [${source.index}] ${this.sourceLabel(source)}`);
                    }
                    lines.push('');
                }
            }

            const blob = new Blob([lines.join('\n')], { type: 'text/markdown;charset=utf-8' });
            const url = URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.href = url;
            const filePrefix = appName.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'jarvis';
            link.download = `${filePrefix}-chat-${new Date().toISOString().slice(0, 10)}.md`;
            document.body.appendChild(link);
            link.click();
            link.remove();
            URL.revokeObjectURL(url);
        },

        sourceLabel(source) {
            if (!source) return '';
            return source.filename + (source.page ? ` p.${source.page}` : '');
        },

        progressLabel(step) {
            const labels = {
                queries: 'Queries',
                search: 'Search',
                fetch: 'Fetch',
                index: 'Index',
                analyze: 'Analyze'
            };
            return labels[step] || step || 'Research';
        },

        openSourcePreview(source) {
            this.sourceModal = { show: true, source };
        },
};
