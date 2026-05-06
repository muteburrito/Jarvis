window.jarvisMessages = {
        async sendMessage() {
            const query = this.input.trim();
            const codeSnippet = this.codeSnippet.trim();
            const imageAttachments = [...this.imageAttachments];
            if ((!query && !codeSnippet && imageAttachments.length === 0) || this.isStreaming) return;

            const displayContent = this.buildUserMessage(query, codeSnippet, imageAttachments);
            this.messages.push({ role: 'user', content: displayContent, attachments: imageAttachments.map(attachment => ({
                name: attachment.name,
                type: attachment.type,
                size: attachment.size,
                previewUrl: attachment.previewUrl
            })) });
            this.input = '';
            this.codeSnippet = '';
            this.imageAttachments = [];
            this.codePanelOpen = false;

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
                    body: JSON.stringify({
                        query: serverQuery,
                        code_snippet: codeSnippet,
                        code_language: this.detectCodeLanguage(codeSnippet),
                        research: this.researchMode,
                        locale: navigator.language || '',
                        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || '',
                        history: this.messages.slice(0, -2).map(m => ({
                            role: m.role,
                            content: m.content
                        }))
                    })
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

