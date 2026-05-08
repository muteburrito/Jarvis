window.jarvisMessageContext = {
        setReplyTo(message, index) {
            if (!message) return;
            this.replyTo = {
                index,
                role: message.role || 'message',
                content: message.content || '',
                excerpt: this.messageExcerpt(message.content || '')
            };
            this.$nextTick(() => {
                if (this.$refs.chatInput) this.$refs.chatInput.focus();
            });
        },

        clearReplyTo() {
            this.replyTo = null;
        },

        normalizedReplyContext() {
            if (!this.replyTo || !this.replyTo.content) return null;
            return {
                role: this.replyTo.role || 'message',
                content: this.replyTo.content
            };
        },

        safeReplyContext() {
            try {
                return this.normalizedReplyContext();
            } catch {
                return null;
            }
        },

        updateMentionState() {
            const match = this.currentMentionMatch();
            this.mentionQuery = match ? match.query : '';
            this.mentionOpen = Boolean(match);
        },

        currentMentionMatch() {
            const input = this.input || '';
            const cursor = this.$refs.chatInput?.selectionStart ?? input.length;
            const prefix = input.slice(0, cursor);
            const match = prefix.match(/(?:^|\s)([@#])([^\s@#]{0,80})$/);
            if (!match) return null;
            return {
                marker: match[1],
                query: match[2] || '',
                start: prefix.length - match[0].trimStart().length,
                end: cursor
            };
        },

        mentionSuggestions() {
            if (!this.mentionOpen) return [];
            const query = (this.mentionQuery || '').toLowerCase();
            const documents = Array.isArray(this.documents) ? this.documents : [];
            const focused = Array.isArray(this.focusedDocuments) ? this.focusedDocuments : [];
            return documents
                .filter(doc => !focused.some(item => item.id === doc.id))
                .filter(doc => {
                    const filename = (doc.filename || '').toLowerCase();
                    const path = (doc.file_path || '').toLowerCase();
                    return !query || filename.includes(query) || path.includes(query);
                })
                .slice(0, 6);
        },

        selectMentionDocument(doc) {
            if (!doc) return;
            this.addFocusDocument(doc);

            const match = this.currentMentionMatch();
            if (match) {
                const before = this.input.slice(0, match.start);
                const after = this.input.slice(match.end);
                this.input = `${before}@${this.documentMentionLabel(doc)} ${after}`.replace(/\s+$/, ' ');
                this.$nextTick(() => {
                    this.autoResize(this.$refs.chatInput);
                    this.$refs.chatInput?.focus();
                });
            }
            this.mentionQuery = '';
            this.mentionOpen = false;
        },

        collectFocusDocuments(query) {
            const focused = Array.isArray(this.focusedDocuments) ? this.focusedDocuments : [];
            const docs = [...focused];
            const seen = new Set(docs.map(doc => doc.id));
            const text = String(query || '').toLowerCase();
            const documents = Array.isArray(this.documents) ? this.documents : [];

            for (const doc of documents) {
                if (seen.has(doc.id)) continue;
                const filename = (doc.filename || '').toLowerCase();
                if (!filename) continue;
                const mentionLabel = this.documentMentionLabel(doc).toLowerCase();
                if (
                    text.includes(`@${filename}`) ||
                    text.includes(`#${filename}`) ||
                    text.includes(`@${mentionLabel}`) ||
                    text.includes(`#${mentionLabel}`)
                ) {
                    docs.push(this.compactDocument(doc));
                    seen.add(doc.id);
                }
            }
            return docs;
        },

        safeFocusDocuments(query) {
            try {
                return this.collectFocusDocuments(query);
            } catch {
                return [];
            }
        },

        addFocusDocument(doc) {
            if (!Array.isArray(this.focusedDocuments)) this.focusedDocuments = [];
            if (!doc || this.focusedDocuments.some(item => item.id === doc.id)) return;
            this.focusedDocuments.push(this.compactDocument(doc));
        },

        removeFocusDocument(id) {
            this.focusedDocuments = this.focusedDocuments.filter(doc => doc.id !== id);
        },

        compactDocument(doc) {
            return {
                id: doc.id,
                filename: doc.filename || doc.file_path || 'indexed file',
                file_path: doc.file_path || ''
            };
        },

        documentMentionLabel(doc) {
            return String(doc?.filename || doc?.file_path || 'file').replace(/\s+/g, '_');
        },
};
