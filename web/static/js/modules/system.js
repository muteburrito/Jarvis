window.jarvisSystem = {
        async loadSystemInfo() {
            try {
                const resp = await fetch(this.apiURL('/api/v1/health'));
                if (resp.ok) {
                    this.systemInfo = await resp.json();
                    if (this.systemInfo?.engine) {
                        this.engineStatus = this.systemInfo.engine;
                    }
                    if (!localStorage.getItem('jarvis:selectedChatModel') && this.systemInfo?.chat_model) {
                        this.selectedChatModel = this.systemInfo.chat_model;
                    }
                }
            } catch {}
        },

        async loadEngineStatus() {
            try {
                const resp = await fetch(this.apiURL('/api/v1/engine'));
                if (resp.ok) {
                    const prevReady = this.engineStatus?.ready;
                    this.engineStatus = await resp.json();
                    if (!prevReady && this.engineStatus?.ready) {
                        this.loadModelOptions();
                        this.loadSystemInfo();
                    }
                }
            } catch {}
        },

        pollEngineIfBusy() {
            if (!this.engineStatus || !this.engineStatus.ready || this.engineStatus.state === 'downloading' || this.engineStatus.state === 'starting' || this.isStreaming) {
                this.loadEngineStatus();
            }
        },

        isEngineBusy() {
            return Boolean(this.engineStatus && (this.engineStatus.state === 'downloading' || this.engineStatus.state === 'starting'));
        },

        engineBackendLabel() {
            const backend = this.engineStatus?.backend || this.systemInfo?.backend || 'llamacpp';
            return backend === 'ollama' ? 'Ollama' : 'On-Device GGUF';
        },

        formatEngineProgress() {
            if (!this.engineStatus) return '';
            const done = Number(this.engineStatus.completed_bytes || 0);
            const total = Number(this.engineStatus.total_bytes || 0);
            if (total > 0 && done >= 0) {
                const doneMB = Math.round(done / (1024 * 1024));
                const totalMB = Math.round(total / (1024 * 1024));
                return `${this.formatMB(doneMB)} / ${this.formatMB(totalMB)} (${this.engineStatus.progress_pct || 0}%)`;
            }
            if (this.engineStatus.progress_pct > 0 && this.engineStatus.progress_pct < 100) {
                return `${this.engineStatus.progress_pct}%`;
            }
            return '';
        },

        async loadModelOptions() {
            try {
                const resp = await fetch(this.apiURL('/api/v1/models'));
                if (resp.ok) {
                    const data = await resp.json();
                    this.availableChatModels = data.models || [];
                    if (data.engine) {
                        this.engineStatus = data.engine;
                    }
                    if (!localStorage.getItem('jarvis:selectedChatModel') && data.default) {
                        this.selectedChatModel = data.default;
                    }
                }
            } catch {}
        },

        gemmaModelOptions() {
            return [
                { value: 'gemma4:e2b', label: 'Gemma 4 2B', hint: 'Fastest, everyday & images' },
                { value: 'gemma4:e4b', label: 'Gemma 4 4B', hint: 'Balanced, documents & images' },
                { value: 'qwen2.5:3b', label: 'Qwen 2.5 3B', hint: 'Fast spreadsheets, data & docs' },
                { value: 'qwen2.5:7b', label: 'Qwen 2.5 7B', hint: 'Best for data, Excel & coding' },
                { value: 'llama3.2:3b', label: 'Llama 3.2 3B', hint: 'Natural writing & summaries' },
                { value: 'gemma4:26b', label: 'Gemma 4 26B', hint: 'Workstation pro (16GB+ VRAM)' }
            ];
        },

        selectedModelLabel() {
            const selected = this.gemmaModelOptions()
                .find(option => this.sameModelName(option.value, this.selectedChatModel));
            return selected?.label || this.selectedChatModel || 'Gemma 4';
        },

        selectChatModel(model) {
            this.selectedChatModel = model;
            localStorage.setItem('jarvis:selectedChatModel', model);
            this.modelMenuOpen = false;
        },

        sameModelName(a, b) {
            return String(a || '').replace(/:latest$/, '') === String(b || '').replace(/:latest$/, '');
        },

        isModelInstalled(model) {
            if (!this.availableChatModels?.length) return false;
            return this.availableChatModels.some(available => this.sameModelName(available, model));
        },

        async loadAppConfig() {
            try {
                const resp = await fetch(this.apiURL('/api/v1/config'));
                if (resp.ok) {
                    this.appConfig = { ...this.appConfig, ...await resp.json() };
                    document.title = this.appConfig.app_name || 'Jarvis';
                }
            } catch {}
        },

        supportHref() {
            if (this.appConfig.support_url) return this.appConfig.support_url;
            if (!this.appConfig.support_email) return '';
            const subject = encodeURIComponent(
                this.appConfig.support_subject || `${this.appConfig.app_name || 'Jarvis'} Support`
            );
            return `mailto:${this.appConfig.support_email}?subject=${subject}`;
        },

        hasSupportContact() {
            return Boolean(this.supportHref());
        },

        async loadHardwareStatus() {
            try {
                const resp = await fetch(this.apiURL('/api/v1/system'));
                if (resp.ok) {
                    this.hwStatus = await resp.json();
                }
            } catch {}
        },

        formatMB(mb) {
            if (!mb || mb <= 0) return 'unknown';
            if (mb >= 1024) {
                return (mb / 1024).toFixed(1).replace(/\.0$/, '') + ' GB';
            }
            return mb + ' MB';
        },
};
