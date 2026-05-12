window.jarvisMessageComposer = {
        createMessageDraft() {
            const query = this.input.trim();
            return {
                id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
                query,
                codeSnippet: this.codeSnippet.trim(),
                imageAttachments: [...this.imageAttachments],
                replyTo: this.safeReplyContext(),
                focusDocuments: this.safeFocusDocuments(query),
                researchMode: Boolean(this.researchMode),
                thinkingMode: Boolean(this.thinkingMode),
                model: this.selectedChatModel || '',
                queuedAt: new Date().toISOString()
            };
        },

        hasDraftContent(draft) {
            return Boolean(
                draft?.query ||
                draft?.codeSnippet ||
                draft?.imageAttachments?.length
            );
        },

        buildOutgoingUserMessage(draft) {
            return {
                role: 'user',
                content: this.buildUserMessage(
                    draft.query,
                    draft.codeSnippet,
                    draft.imageAttachments
                ),
                attachments: draft.imageAttachments.map(attachment => ({
                    name: attachment.name,
                    type: attachment.type,
                    size: attachment.size,
                    previewUrl: attachment.previewUrl
                })),
                replyTo: draft.replyTo,
                focusedDocuments: draft.focusDocuments
            };
        },

        buildChatPayload(query, draft) {
            return {
                query,
                code_snippet: draft.codeSnippet,
                code_language: this.detectCodeLanguage(draft.codeSnippet),
                research: draft.researchMode,
                thinking: draft.thinkingMode,
                model: draft.model,
                reply_to: draft.replyTo,
                focus_document_ids: draft.focusDocuments.map(doc => doc.id),
                focus_files: draft.focusDocuments.map(doc => doc.filename),
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

        resetComposerHeight() {
            if (this.$refs.chatInput) {
                this.$refs.chatInput.style.height = 'auto';
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

        messageExcerpt(content) {
            const text = String(content || '')
                .replace(/```[\s\S]*?```/g, '[code snippet]')
                .replace(/\s+/g, ' ')
                .trim();
            if (text.length <= 160) return text;
            return text.slice(0, 157) + '...';
        },
};
