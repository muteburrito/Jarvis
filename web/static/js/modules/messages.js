window.jarvisMessages = {
        async sendMessage() {
            const query = this.input.trim();
            const codeSnippet = this.codeSnippet.trim();
            const imageAttachments = [...this.imageAttachments];
            if ((!query && !codeSnippet && imageAttachments.length === 0) || this.isStreaming) return;

            const replyTo = this.safeReplyContext();
            const focusDocuments = this.safeFocusDocuments(query);
            this.messages.push(this.buildOutgoingUserMessage(
                query,
                codeSnippet,
                imageAttachments,
                replyTo,
                focusDocuments
            ));
            this.resetComposerState();

            if (this.$refs.chatInput) {
                this.$refs.chatInput.style.height = 'auto';
            }

            this.isStreaming = true;
            this.messages.push({ role: 'assistant', content: '', sources: [], progress: [] });
            const assistantIdx = this.messages.length - 1;
            this.scrollToBottom(true);

            let pendingTokenText = '';
            let lastFlushAt = 0;
            const flushTokens = (force = false) => {
                if (!pendingTokenText) return;
                const now = Date.now();
                if (!force && now - lastFlushAt < 50) return;
                this.messages[assistantIdx].content += pendingTokenText;
                pendingTokenText = '';
                lastFlushAt = now;
                this.scrollToBottom(false);
            };

            try {
                const uploadedImages = [];
                for (const attachment of imageAttachments) {
                    const result = await this.uploadFile(attachment.file, attachment.uploadName, true);
                    if (result?.filename) uploadedImages.push(result.filename);
                }
                if (uploadedImages.length > 0) {
                    this.showToast(`Indexed ${uploadedImages.length} pasted image${uploadedImages.length === 1 ? '' : 's'}`);
                }
                const serverQuery = this.buildServerQuery(query, uploadedImages);
                const response = await fetch(this.apiURL('/api/v1/chat'), {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(this.buildChatPayload(
                        serverQuery,
                        codeSnippet,
                        replyTo,
                        focusDocuments
                    ))
                });

                if (!response.ok) {
                    const err = await response.json();
                    throw new Error(err.error || 'Request failed');
                }

                const reader = response.body.getReader();
                const decoder = new TextDecoder();
                let buffer = '';

                while (true) {
                    const { done, value } = await reader.read();
                    if (done) break;

                    buffer += decoder.decode(value, { stream: true });
                    const lines = buffer.split('\n');
                    buffer = lines.pop();

                    for (const line of lines) {
                        if (line.startsWith('data: ')) {
                            try {
                                const data = JSON.parse(line.slice(6));
                                if (data.progress) {
                                    this.messages[assistantIdx].progress.push(data.progress);
                                    this.scrollToBottom(false);
                                } else if (data.done) {
                                    flushTokens(true);
                                    this.messages[assistantIdx].sources = data.sources || [];
                                } else if (data.token) {
                                    pendingTokenText += data.token;
                                    flushTokens(false);
                                }
                            } catch {}
                        }
                    }
                }
                flushTokens(true);
            } catch (error) {
                flushTokens(true);
                const errorText = 'Error: ' + error.message;
                if (this.messages[assistantIdx].content.trim()) {
                    this.messages[assistantIdx].content += '\n\n_' + errorText + '_';
                } else {
                    this.messages[assistantIdx].content = errorText;
                }
                this.showToast(error.message, 'error');
            } finally {
                this.isStreaming = false;
                await this.saveActiveChat();
                this.scrollToBottom(false);
                this.$nextTick(() => {
                    if (this.$refs.chatInput) this.$refs.chatInput.focus();
                });
            }
        },

        buildOutgoingUserMessage(query, codeSnippet, imageAttachments, replyTo, focusDocuments) {
            return {
                role: 'user',
                content: this.buildUserMessage(query, codeSnippet, imageAttachments),
                attachments: imageAttachments.map(attachment => ({
                    name: attachment.name,
                    type: attachment.type,
                    size: attachment.size,
                    previewUrl: attachment.previewUrl
                })),
                replyTo,
                focusedDocuments: focusDocuments
            };
        },

        buildChatPayload(query, codeSnippet, replyTo, focusDocuments) {
            return {
                query,
                code_snippet: codeSnippet,
                code_language: this.detectCodeLanguage(codeSnippet),
                research: this.researchMode,
                reply_to: replyTo,
                focus_document_ids: focusDocuments.map(doc => doc.id),
                focus_files: focusDocuments.map(doc => doc.filename),
                locale: navigator.language || '',
                timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || '',
                history: this.messages.slice(0, -2).map(message => ({
                    role: message.role,
                    content: message.content
                }))
            };
        },

        resetComposerState() {
            this.input = '';
            this.codeSnippet = '';
            this.imageAttachments = [];
            this.replyTo = null;
            this.focusedDocuments = [];
            this.mentionQuery = '';
            this.mentionOpen = false;
            this.codePanelOpen = false;
        },

        buildServerQuery(query, uploadedImages = []) {
            const imageCount = uploadedImages.length;
            const base = query || (imageCount > 0 ? 'Please describe the attached image.' : '');
            if (imageCount === 0) return base;
            const names = uploadedImages.map(name => `- ${name}`).join('\n');
            return [
                base,
                '',
                `Attached image${imageCount === 1 ? '' : 's'} just indexed for this question:`,
                names,
                '',
                'Use the image descriptions from the indexed context when answering.'
            ].join('\n');
        },

        buildUserMessage(query, codeSnippet, imageAttachments = []) {
            const hasImages = imageAttachments.length > 0;
            if (!codeSnippet) return query || (hasImages ? 'Please describe the attached image.' : '');
            const intro = query || (
                hasImages
                    ? 'Please analyze this code snippet and attached image.'
                    : 'Please analyze this code snippet.'
            );
            const language = this.detectCodeLanguage(codeSnippet);
            return `${intro}\n\n\`\`\`${language}\n${codeSnippet}\n\`\`\``;
        },

        setReplyTo(message, index) {
            if (!message || this.isStreaming) return;
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

        messageExcerpt(content) {
            const text = String(content || '')
                .replace(/```[\s\S]*?```/g, '[code snippet]')
                .replace(/\s+/g, ' ')
                .trim();
            if (text.length <= 160) return text;
            return text.slice(0, 157) + '...';
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

        handleChatPaste(event) {
            if (this.isStreaming) return;
            const items = Array.from(event.clipboardData?.items || []);
            const imageItems = items.filter(item => item.kind === 'file' && item.type.startsWith('image/'));
            if (imageItems.length === 0) return;
            event.preventDefault();
            for (const item of imageItems) {
                const file = item.getAsFile();
                if (file) this.addImageAttachment(file);
            }
        },

        handleChatDrop(event) {
            if (this.isStreaming) return;
            const files = Array.from(event.dataTransfer?.files || []);
            if (files.length === 0) return;
            event.preventDefault();
            const imageFiles = files.filter(file => file.type.startsWith('image/'));
            const otherFiles = files.filter(file => !file.type.startsWith('image/'));
            for (const file of imageFiles) {
                this.addImageAttachment(file);
            }
            if (otherFiles.length > 0) {
                this.uploadFiles(otherFiles);
            }
        },

        addImageAttachment(file) {
            if (!file || !file.type.startsWith('image/')) {
                this.showToast('Only images can be pasted into chat', 'error');
                return;
            }
            const ext = this.imageExtension(file.type);
            const name = file.name && file.name !== 'image.png'
                ? file.name
                : `pasted-image-${Date.now()}-${this.imageAttachments.length + 1}.${ext}`;
            const uploadName = name.replace(/[^\w.\-]+/g, '_');
            this.imageAttachments.push({
                id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
                file,
                name,
                uploadName,
                type: file.type,
                size: file.size,
                previewUrl: URL.createObjectURL(file)
            });
        },

        removeImageAttachment(id) {
            const index = this.imageAttachments.findIndex(attachment => attachment.id === id);
            if (index === -1) return;
            const [attachment] = this.imageAttachments.splice(index, 1);
            if (attachment?.previewUrl) URL.revokeObjectURL(attachment.previewUrl);
        },

        imageExtension(type) {
            const map = {
                'image/jpeg': 'jpg',
                'image/png': 'png',
                'image/webp': 'webp',
                'image/gif': 'gif',
                'image/bmp': 'bmp'
            };
            return map[type] || 'png';
        },
};

