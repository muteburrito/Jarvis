window.jarvisSystem = {
        async loadSystemInfo() {
            try {
                const resp = await fetch(this.apiURL('/api/v1/health'));
                if (resp.ok) {
                    this.systemInfo = await resp.json();
                }
            } catch {}
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
