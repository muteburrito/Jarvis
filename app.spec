# PyInstaller spec file
block_cipher = None

a = Analysis(
    ['app.py'],              # The main script that PyInstaller should bundle
    pathex=['.'],             # List of paths to search for imports and modules
    binaries=[],              # Extra binary files to bundle (e.g., `.dll`, `.so`)
    datas=[                   # List of data files to include (e.g., templates, static files)
        ('templates', 'templates'),
        ('static', 'static'),
        ('lang_funcs.py', '.'),  # lang_chain.py
        ('utils.py', '.') # utils.py
    ],
    hiddenimports=['utils'],         # Any modules that PyInstaller might miss during analysis
    hookspath=[],             # Paths to custom hook files, if any
    runtime_hooks=[],         # Hooks that modify the application at runtime, usually empty
    excludes=[],              # Any modules you explicitly want to exclude from the build
    win_no_prefer_redirects=False,  # Windows-specific, used to avoid redirecting Windows API calls
    win_private_assemblies=False,   # Whether to bundle private assemblies (leave it False)
    cipher=block_cipher        # Encryption setting for bytecode (set to None, no encryption)
)

pyz = PYZ(a.pure, a.zipped_data, cipher=block_cipher)

exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.zipfiles,
    a.datas,
    strip=False,
    upx=True,  # UPX compression
    name='Jarvis', 
    console=True,  # To see the terminal for logs and debugging
    icon='jarvis_logo.ico'
)
