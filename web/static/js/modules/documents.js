window.jarvisDocuments = {
        async loadDocuments() {
            try {
                const resp = await fetch(this.apiURL('/api/v1/documents'));
                if (resp.ok) {
                    this.documents = await resp.json();
                }
            } catch {}
        },

        async uploadFile(file, uploadName = '', quiet = false) {
            if (!file) return;

            this.isUploading = true;
            this.uploadProgress = 0;
            this.uploadingName = uploadName || file.name;

            const formData = new FormData();
            formData.append('file', file, uploadName || file.name);

            try {
                const xhr = new XMLHttpRequest();
                const result = await new Promise((resolve, reject) => {
                    xhr.upload.addEventListener('progress', (e) => {
                        if (e.lengthComputable) {
                            this.uploadProgress = Math.round((e.loaded / e.total) * 90);
                        }
                    });
                    xhr.addEventListener('load', () => {
                        if (xhr.status >= 200 && xhr.status < 300) {
                            resolve(JSON.parse(xhr.responseText));
                        } else {
                            try {
                                reject(new Error(JSON.parse(xhr.responseText).error));
                            } catch {
                                reject(new Error('Upload failed'));
                            }
                        }
                    });
                    xhr.addEventListener('error', () => reject(new Error('Network error')));
                    xhr.open('POST', this.apiURL('/api/v1/upload'));
                    xhr.send(formData);
                });

                this.uploadProgress = 100;
                if (!quiet) {
                    this.showToast(`${result.filename} processed (${result.chunks} chunks)`);
                }
                await this.loadDocuments();
                return result;
            } catch (error) {
                this.showToast(error.message, 'error');
                throw error;
            } finally {
                setTimeout(() => {
                    this.isUploading = false;
                    this.uploadProgress = 0;
                    this.uploadingName = '';
                }, 500);
                if (this.$refs.fileInput) this.$refs.fileInput.value = '';
                if (this.$refs.folderInput) this.$refs.folderInput.value = '';
            }
        },

        async uploadFiles(files) {
            const allFiles = Array.from(files || []);
            const list = allFiles.filter(file => !this.isLikelyBinaryFile(file));
            if (list.length === 0) {
                if (allFiles.length > 0) {
                    this.showToast('No text-like files found', 'error');
                }
                return;
            }
            if (list.length === 1) {
                await this.uploadFile(list[0]);
                return;
            }

            let processed = 0;
            for (const file of list) {
                const relativePath = file.webkitRelativePath || file.name;
                const uploadName = relativePath.replace(/[\\/]+/g, '__');
                await this.uploadFile(file, uploadName, true);
                processed++;
            }
            this.showToast(`Indexed ${processed} files from folder`);
            await this.loadDocuments();
        },

        isLikelyBinaryFile(file) {
            const name = (file?.name || '').toLowerCase();
            const binaryExtensions = [
                '.exe', '.dll', '.so', '.dylib', '.bin', '.obj', '.o', '.a', '.lib',
                '.class', '.jar', '.war', '.zip', '.7z', '.rar', '.tar', '.gz', '.bz2',
                '.xz', '.iso', '.msi', '.pkg', '.dmg', '.db', '.sqlite', '.sqlite3',
                '.wasm', '.pdb', '.ilk', '.cache', '.lockb', '.pyc'
            ];
            return binaryExtensions.some(ext => name.endsWith(ext));
        },

        async deleteDocument(id) {
            try {
                const resp = await fetch(this.apiURL(`/api/v1/documents/${id}`), {
                    method: 'DELETE'
                });
                if (resp.ok) {
                    this.showToast('Document removed');
                    await this.loadDocuments();
                } else {
                    const err = await resp.json();
                    this.showToast(err.error || 'Delete failed', 'error');
                }
            } catch (error) {
                this.showToast('Failed to delete document', 'error');
            }
        },

        async clearAll() {
            if (!confirm('Clear all documents and embeddings? This cannot be undone.')) return;
            this.isClearing = true;
            try {
                const resp = await fetch(this.apiURL('/api/v1/documents'), {
                    method: 'DELETE'
                });
                if (resp.ok) {
                    this.documents = [];
                    this.showToast('All documents and embeddings cleared');
                } else {
                    const err = await resp.json();
                    this.showToast(err.error || 'Failed to clear', 'error');
                }
            } catch (error) {
                this.showToast('Failed to clear documents', 'error');
            } finally {
                this.isClearing = false;
            }
        },

        handleDrop(event) {
            this.dragOver = false;
            const files = event.dataTransfer.files;
            if (files.length > 0) {
                this.uploadFiles(files);
            }
        },
};

