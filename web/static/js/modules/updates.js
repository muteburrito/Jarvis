window.jarvisUpdates = {
        async loadUpdateStatus() {
            try {
                const resp = await fetch('/api/v1/update');
                if (resp.ok) {
                    this.updateInfo = await resp.json();
                }
            } catch {}
        },

        async applyUpdate() {
            if (this.updateApplying) return;
            this.updateApplying = true;
            try {
                const resp = await fetch('/api/v1/update/apply', { method: 'POST' });
                if (!resp.ok) {
                    const err = await resp.json();
                    throw new Error(err.error || 'Update failed');
                }
                this.waitForRestart();
            } catch (error) {
                this.showToast('Update failed: ' + error.message, 'error');
                this.updateApplying = false;
            }
        },

        openNotesModal(tag, content) {
            this.notesModal = { show: true, tag: tag || '', content: content || '' };
        },

        waitForRestart() {
            const poll = () => {
                fetch('/api/v1/health')
                    .then(r => { if (r.ok) window.location.reload(); else setTimeout(poll, 1500); })
                    .catch(() => setTimeout(poll, 1500));
            };
            setTimeout(poll, 3000);
        },

};

