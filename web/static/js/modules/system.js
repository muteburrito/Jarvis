window.jarvisSystem = {
        async loadSystemInfo() {
            try {
                const resp = await fetch(this.apiURL('/api/v1/health'));
                if (resp.ok) {
                    this.systemInfo = await resp.json();
                    if (!localStorage.getItem('jarvis:selectedChatModel') && this.systemInfo?.chat_model) {
                        this.selectedChatModel = this.systemInfo.chat_model;
                    }
                }
            } catch {}
        },

        async loadModelOptions() {
            try {
                const resp = await fetch(this.apiURL('/api/v1/models'));
                if (resp.ok) {
                    const data = await resp.json();
                    this.availableChatModels = data.models || [];
                    if (!localStorage.getItem('jarvis:selectedChatModel') && data.default) {
                        this.selectedChatModel = data.default;
                    }
                }
            } catch {}
        },

        gemmaModelOptions() {
            return [
                { value: 'gemma4:e2b', label: 'Gemma 4 2B', hint: 'fastest' },
                { value: 'gemma4:e4b', label: 'Gemma 4 4B', hint: 'default' },
                { value: 'gemma4:26b', label: 'Gemma 4 26B', hint: 'workstation' }
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
            if (!this.availableChatModels?.length) return true;
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
