window.jarvisMessageQueue = {
        enqueueMessageDraft(draft) {
            this.messageQueue.push(draft);
            this.showToast(`Queued follow-up ${this.messageQueue.length}`);
            this.scrollToBottom(false);
            this.$nextTick(() => {
                if (this.$refs.chatInput) this.$refs.chatInput.focus();
            });
        },

        processNextQueuedMessage() {
            if (this.isStreaming || this.editingQueueID || this.messageQueue.length === 0) return;
            const nextDraft = this.messageQueue.shift();
            this.$nextTick(() => this.submitMessageDraft(nextDraft));
        },

        removeQueuedMessage(id) {
            this.messageQueue = this.messageQueue.filter(item => item.id !== id);
            if (this.editingQueueID === id) {
                this.cancelEditQueuedMessage();
            }
        },

        startEditQueuedMessage(draft) {
            if (!draft) return;
            this.editingQueueID = draft.id;
            this.queueEditText = draft.query || '';
            this.$nextTick(() => {
                const editor = document.querySelector(`[data-queue-editor="${draft.id}"]`);
                editor?.focus();
            });
        },

        saveQueuedMessage(id) {
            const draft = this.messageQueue.find(item => item.id === id);
            if (!draft) return;
            const query = this.queueEditText.trim();
            if (!query && !draft.codeSnippet && draft.imageAttachments.length === 0) {
                this.removeQueuedMessage(id);
                return;
            }
            draft.query = query;
            this.cancelEditQueuedMessage();
            this.processNextQueuedMessage();
        },

        cancelEditQueuedMessage() {
            this.editingQueueID = '';
            this.queueEditText = '';
            this.processNextQueuedMessage();
        },

        moveQueuedMessage(id, direction) {
            const index = this.messageQueue.findIndex(item => item.id === id);
            if (index === -1) return;
            const nextIndex = index + direction;
            if (nextIndex < 0 || nextIndex >= this.messageQueue.length) return;
            const queue = [...this.messageQueue];
            const [draft] = queue.splice(index, 1);
            queue.splice(nextIndex, 0, draft);
            this.messageQueue = queue;
        },

        queuedMessageLabel(draft) {
            if (!draft) return '';
            const text = draft.query || (
                draft.codeSnippet
                    ? 'Code snippet follow-up'
                    : 'Image follow-up'
            );
            return this.messageExcerpt(text);
        },
};
