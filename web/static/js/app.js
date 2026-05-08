function chatApp() {
    return Object.assign({
        messages: [],
        input: '',
        isStreaming: false,
        sidebarOpen: window.innerWidth >= 1024,
        documents: [],
        chatSessions: [],
        activeChatID: '',
        showDocumentsPanel: false,
        showWorkbenchPanel: false,
        taskState: null,
        uploadProgress: 0,
        isUploading: false,
        uploadingName: '',
        dragOver: false,
        isClearing: false,
        researchMode: false,
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
        showRoadmap: false,
        promptTemplates: [
            {
                name: 'Summarize document',
                prompt: [
                    'Summarize this document.',
                    'Focus on the main points, decisions, risks, and follow-up actions.'
                ].join(' ')
            },
            {
                name: 'Code review',
                prompt: [
                    'Review this code like a senior staff engineer.',
                    'Call out bugs, edge cases, design risks, and missing tests.',
                    'Keep the feedback practical.'
                ].join(' ')
            },
            {
                name: 'Find TODOs',
                prompt: [
                    'Find all TODOs, FIXMEs, incomplete work, and risky placeholders.',
                    'Group them by file or topic and suggest the next action for each one.'
                ].join(' ')
            },
            {
                name: 'Release notes',
                prompt: [
                    'Generate release notes from the available git log or project notes.',
                    'Group changes into features, fixes, and known risks.'
                ].join(' ')
            }
        ],

        init() {
            this.loadChats();
            this.loadAppConfig();
            this.loadDocuments();
            this.loadTaskState();
            this.loadSystemInfo();
            this.loadHardwareStatus();
            this.loadUpdateStatus();
            setInterval(() => this.loadUpdateStatus(), 60 * 60 * 1000);

            marked.setOptions({
                breaks: true,
                gfm: true,
            });
            const renderer = new marked.Renderer();
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
                if (!button) return;
                this.copyCodeBlock(button);
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
