window.jarvisChats = {
        chatTitle(chat) {
            return chat?.title || 'New chat';
        },

        chatTimestamp(chat) {
            if (!chat?.updated_at) return '';
            try {
                return new Date(chat.updated_at).toLocaleString([], {
                    month: 'short',
                    day: 'numeric',
                    hour: '2-digit',
                    minute: '2-digit'
                });
            } catch {
                return '';
            }
        },

        currentChatTitle() {
            const active = this.chatSessions.find(chat => chat.id === this.activeChatID);
            return active ? active.title : 'New chat';
        },

        async loadChats() {
            try {
                const resp = await fetch('/api/v1/chats');
                if (!resp.ok) {
                    this.loadLegacyChat();
                    return;
                }
                this.chatSessions = await resp.json();
                if (this.chatSessions.length === 0) {
                    await this.newChat(false);
                    return;
                }
                await this.openChat(this.chatSessions[0].id);
            } catch {
                this.loadLegacyChat();
            }
        },

        loadLegacyChat() {
            const saved = localStorage.getItem('jarvis_messages');
            if (saved) {
                try { this.messages = JSON.parse(saved); } catch {}
            }
        },

        async newChat(showToast = true) {
            await this.saveActiveChat();
            try {
                const resp = await fetch('/api/v1/chats', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ title: 'New chat' })
                });
                if (!resp.ok) throw new Error('Create chat failed');
                const session = await resp.json();
                this.activeChatID = session.id;
                this.messages = session.messages || [];
                await this.refreshChatList();
                if (showToast) this.showToast('New chat created');
                this.$nextTick(() => {
                    if (this.$refs.chatInput) this.$refs.chatInput.focus();
                });
            } catch {
                this.activeChatID = '';
                this.messages = [];
                localStorage.removeItem('jarvis_messages');
            }
        },

        async openChat(id) {
            if (!id || this.isStreaming) return;
            if (this.activeChatID && this.activeChatID !== id) {
                await this.saveActiveChat();
            }
            try {
                const resp = await fetch(`/api/v1/chats/${id}`);
                if (!resp.ok) throw new Error('Chat not found');
                const session = await resp.json();
                this.activeChatID = session.id;
                this.messages = session.messages || [];
                this.scrollToBottom(true);
            } catch {
                this.showToast('Failed to open chat', 'error');
            }
        },

        async deleteChat(id) {
            if (!id || this.isStreaming) return;
            try {
                const resp = await fetch(`/api/v1/chats/${id}`, { method: 'DELETE' });
                if (!resp.ok) throw new Error('Delete failed');
                await this.refreshChatList();
                if (this.activeChatID === id) {
                    if (this.chatSessions.length > 0) {
                        await this.openChat(this.chatSessions[0].id);
                    } else {
                        await this.newChat(false);
                    }
                }
            } catch {
                this.showToast('Failed to delete chat', 'error');
            }
        },

        async refreshChatList() {
            try {
                const resp = await fetch('/api/v1/chats');
                if (resp.ok) this.chatSessions = await resp.json();
            } catch {}
        },

        async saveActiveChat() {
            if (!this.activeChatID) return;
            const firstUser = this.messages.find(message => message.role === 'user' && message.content);
            const title = firstUser ? firstUser.content : 'New chat';
            try {
                await fetch(`/api/v1/chats/${this.activeChatID}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        title,
                        messages: this.messages.slice(-100).map(message => ({
                            role: message.role,
                            content: message.content || '',
                            sources: message.sources || [],
                            progress: message.progress || [],
                            attachments: (message.attachments || []).map(attachment => ({
                                name: attachment.name || 'image',
                                type: attachment.type || 'image',
                                size: attachment.size || 0
                            }))
                        }))
                    })
                });
                await this.refreshChatList();
            } catch {
                try { localStorage.setItem('jarvis_messages', JSON.stringify(this.messages.slice(-100))); } catch {}
            }
        },
};

