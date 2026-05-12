function chatApp() {
    return Object.assign({
        messages: [],
        input: '',
        isStreaming: false,
        messageQueue: [],
        editingQueueID: '',
        queueEditText: '',
        clockTick: Date.now(),
        sidebarOpen: window.innerWidth >= 1024,
        documents: [],
        chatSessions: [],
        activeChatID: '',
        showDocumentsPanel: false,
        showWorkbenchPanel: false,
        taskState: null,
        repoMap: null,
        projectState: null,
        projectActivity: null,
        diffSummary: null,
        expandedDiffs: {},
        toolSearch: '',
        toolResults: [],
        toolPreview: null,
        toolLoading: false,
        commandPreset: 'git status --short',
        commandResult: null,
        commandRunning: false,
        repoSearch: '',
        uploadProgress: 0,
        isUploading: false,
        uploadingName: '',
        dragOver: false,
        isClearing: false,
        researchMode: false,
        thinkingMode: false,
        modelMenuOpen: false,
        selectedChatModel: localStorage.getItem('jarvis:selectedChatModel') || 'gemma4:e4b',
        availableChatModels: [],
        showHelp: false,
        codePanelOpen: false,
        codeSnippet: '',
        imageAttachments: [],
        replyTo: null,
        focusedDocuments: [],
        mentionQuery: '',
        mentionOpen: false,
        systemInfo: null,
        hwStatus: null,
        appConfig: {
            app_name: 'Jarvis',
            support_email: '',
            support_subject: 'Jarvis Support',
            support_url: ''
        },
        toast: { show: false, message: '', type: 'success' },
        updateInfo: null,
        updateDismissed: false,
        updateApplying: false,
        notesModal: { show: false, tag: '', content: '' },
        sourceModal: { show: false, source: null },
        webSourceModal: { show: false, url: '', title: '' },
        showRoadmap: false,

        init() {
            this.loadChats();
            this.loadAppConfig();
            this.loadDocuments();
            this.loadTaskState();
            this.loadSystemInfo();
            this.loadModelOptions();
            this.loadHardwareStatus();
            this.loadUpdateStatus();
            setInterval(() => this.loadUpdateStatus(), 60 * 60 * 1000);
            setInterval(() => { this.clockTick = Date.now(); }, 1000);

            marked.setOptions({
                breaks: true,
                gfm: true,
            });
            const renderer = new marked.Renderer();
            renderer.link = (href, title, text) => {
                if (typeof href === 'object' && href !== null) {
                    title = href.title || '';
                    text = href.text || href.href || '';
                    href = href.href || '';
                }
                const safeHref = this.escapeHtml(href || '');
                const safeTitle = title ? ` title="${this.escapeHtml(title)}"` : '';
                const safeText = text || safeHref;
                const isExternal = /^https?:\/\//i.test(href || '');
                const targetAttrs = isExternal ? ' target="_blank" rel="noopener noreferrer"' : '';
                return `<a href="${safeHref}"${safeTitle}${targetAttrs}>${safeText}</a>`;
            };
            renderer.code = (code, infoString) => {
                if (typeof code === 'object' && code !== null) {
                    infoString = code.lang || '';
                    code = code.text || '';
                }
                const language = this.normalizeHighlightLanguage((infoString || '').split(/\s+/)[0]);
                const highlighted = this.highlightCode(code, language);
                const className = language ? `hljs language-${language}` : 'hljs';
                const label = language ? this.escapeHtml(language) : 'code';
                return [
                    '<div class="code-block-wrapper">',
                    '<div class="code-block-toolbar">',
                    `<span>${label}</span>`,
                    '<button type="button" class="code-copy-button">Copy</button>',
                    '</div>',
                    `<pre><code class="${className}">${highlighted}</code></pre>`,
                    '</div>'
                ].join('');
            };
            marked.use({ renderer });

            document.addEventListener('click', (event) => {
                const button = event.target.closest('.code-copy-button');
                if (button) {
                    this.copyCodeBlock(button);
                    return;
                }

                const link = event.target.closest('.chat-message a[href]');
                if (!link || !/^https?:\/\//i.test(link.href)) return;
                event.preventDefault();
                this.openWebSource(link.href, link.textContent || link.href);
            });
        },

        apiURL(path) {
            const apiBase = window.jarvisApiBase || '';
            return apiBase + path;
        },
    },
        window.jarvisUi,
        window.jarvisChats,
        window.jarvisMessages,
        window.jarvisSystem,
        window.jarvisDocuments,
        window.jarvisUpdates,
        window.jarvisWorkbench,
    );
}
