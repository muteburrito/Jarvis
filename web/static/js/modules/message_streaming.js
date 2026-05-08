window.jarvisMessageStreaming = {
        async sendMessage() {
            const draft = this.createMessageDraft();
            if (!this.hasDraftContent(draft)) return;

            this.resetComposerState();
            this.resetComposerHeight();

            if (this.isStreaming) {
                this.enqueueMessageDraft(draft);
                return;
            }

            await this.submitMessageDraft(draft);
        },

        async submitMessageDraft(draft) {
            this.messages.push(this.buildOutgoingUserMessage(draft));
            this.isStreaming = true;
            const startedAt = new Date().toISOString();
            this.messages.push({
                role: 'assistant',
                content: '',
                sources: [],
                progress: [],
                startedAt
            });

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
                const uploadedImages = await this.uploadDraftImages(draft.imageAttachments);
                const serverQuery = this.buildServerQuery(draft.query, uploadedImages);
                const response = await fetch(this.apiURL('/api/v1/chat'), {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(this.buildChatPayload(serverQuery, draft))
                });

                if (!response.ok) {
                    const err = await response.json();
                    throw new Error(err.error || 'Request failed');
                }

                await this.readChatStream(response, assistantIdx, flushTokens, token => {
                    pendingTokenText += token;
                });
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
                const assistant = this.messages[assistantIdx];
                if (assistant) {
                    assistant.completedAt = new Date().toISOString();
                    assistant.durationMs = this.responseDurationMs(assistant);
                }
                this.isStreaming = false;
                await this.saveActiveChat();
                this.scrollToBottom(false);
                this.$nextTick(() => {
                    if (this.$refs.chatInput) this.$refs.chatInput.focus();
                });
                this.processNextQueuedMessage();
            }
        },

        async readChatStream(response, assistantIdx, flushTokens, appendToken) {
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
                    if (!line.startsWith('data: ')) continue;
                    try {
                        const data = JSON.parse(line.slice(6));
                        if (data.progress) {
                            this.messages[assistantIdx].progress.push(data.progress);
                            this.scrollToBottom(false);
                        } else if (data.done) {
                            flushTokens(true);
                            this.messages[assistantIdx].sources = data.sources || [];
                        } else if (data.token) {
                            appendToken(data.token);
                            flushTokens(false);
                        }
                    } catch {}
                }
            }
        },
};
