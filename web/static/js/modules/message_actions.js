window.jarvisMessageActions = {
        async copyResponse(message) {
            const content = message?.content || '';
            if (!content.trim()) return;
            try {
                await navigator.clipboard.writeText(content);
                this.showToast('Response copied');
            } catch {
                this.showToast('Could not copy response', 'error');
            }
        },

        async rateResponse(message, rating) {
            if (!message || message.role !== 'assistant') return;
            message.rating = message.rating === rating ? '' : rating;
            await this.saveActiveChat();
        },

        async forkResponse(index) {
            if (index < 0 || index >= this.messages.length) return;
            const messages = this.cloneMessagesForFork(this.messages.slice(0, index + 1));
            const titleSource = messages.find(message => message.role === 'user' && message.content);
            const title = titleSource ? `Fork: ${this.messageExcerpt(titleSource.content)}` : 'Forked chat';

            try {
                await this.saveActiveChat();
                const resp = await fetch(this.apiURL('/api/v1/chats'), {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ title })
                });
                if (!resp.ok) throw new Error('Create fork failed');
                const session = await resp.json();

                const saveResp = await fetch(this.apiURL(`/api/v1/chats/${session.id}`), {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ title, messages })
                });
                if (!saveResp.ok) throw new Error('Save fork failed');
                const savedSession = await saveResp.json();

                this.activeChatID = savedSession.id;
                this.messages = savedSession.messages || messages;
                this.messageQueue = [];
                this.cancelEditQueuedMessage();
                await this.refreshChatList();
                if (!this.chatSessions.some(chat => chat.id === savedSession.id)) {
                    this.chatSessions = [{
                        id: savedSession.id,
                        title: savedSession.title || title,
                        message_count: this.messages.length,
                        updated_at: savedSession.updated_at || new Date().toISOString()
                    }, ...this.chatSessions];
                }
                this.scrollToBottom(true);
                this.showToast('Forked conversation');
            } catch (error) {
                this.showToast(error.message || 'Failed to fork response', 'error');
            }
        },

        cloneMessagesForFork(messages) {
            return messages.map(message => ({
                role: message.role,
                content: message.content || '',
                sources: message.sources || [],
                progress: message.progress || [],
                replyTo: message.replyTo || null,
                focusedDocuments: message.focusedDocuments || [],
                attachments: (message.attachments || []).map(attachment => ({
                    name: attachment.name || 'image',
                    type: attachment.type || 'image',
                    size: attachment.size || 0
                })),
                startedAt: message.startedAt || '',
                completedAt: message.completedAt || '',
                durationMs: message.durationMs || 0,
                rating: message.rating || ''
            }));
        },

        responseDurationMs(message) {
            if (!message?.startedAt) return 0;
            const start = new Date(message.startedAt).getTime();
            if (!Number.isFinite(start)) return 0;
            const end = message.completedAt
                ? new Date(message.completedAt).getTime()
                : this.clockTick;
            if (!Number.isFinite(end) || end < start) return 0;
            return end - start;
        },

        responseDurationLabel(message, index) {
            const isActive = this.isStreaming && index === this.messages.length - 1;
            const duration = this.responseDurationMs(message);
            const label = this.formatDuration(duration);
            return isActive ? `Working for ${label}` : `Took ${label}`;
        },
};
