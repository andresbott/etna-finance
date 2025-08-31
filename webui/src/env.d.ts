interface ImportMetaEnv {
    readonly VITE_SERVER_URL_V0: string
    readonly VITE_AUTH_PATH: string
    // add any other env vars here
}

interface ImportMeta {
    readonly env: ImportMetaEnv
}
