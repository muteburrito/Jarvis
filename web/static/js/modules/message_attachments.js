window.jarvisMessageAttachments = {
        async uploadDraftImages(imageAttachments) {
            const uploadedImages = [];
            for (const attachment of imageAttachments || []) {
                const result = await this.uploadFile(attachment.file, attachment.uploadName, true);
                if (result?.filename) uploadedImages.push(result.filename);
            }
            if (uploadedImages.length > 0) {
                this.showToast(`Indexed ${uploadedImages.length} pasted image${uploadedImages.length === 1 ? '' : 's'}`);
            }
            return uploadedImages;
        },

        handleChatPaste(event) {
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
            const files = Array.from(event.dataTransfer?.files || []);
            if (files.length === 0) return;
            event.preventDefault();
            const imageFiles = files.filter(file => file.type.startsWith('image/'));
            const otherFiles = files.filter(file => !file.type.startsWith('image/'));
            for (const file of imageFiles) {
                this.addImageAttachment(file);
            }
            if (otherFiles.length > 0 && !this.isStreaming) {
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
