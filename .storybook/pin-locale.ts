// Imported first by preview.tsx. The app reads its language from localStorage when i18n starts, and browsers that run
// story files side by side share it, so a story that switched to Russian could decide the language of another file.
try {
    localStorage.setItem('savvy.locale', 'en')
} catch {
    // No storage: the browser's own language is used.
}
