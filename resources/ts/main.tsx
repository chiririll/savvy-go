import { i18nReady } from '@/lib/i18n'
import '@/hooks/use-theme'
import React from 'react'
import ReactDOM from 'react-dom/client'
import { App } from './app/App'
import './index.css'

void i18nReady.finally(() => {
    ReactDOM.createRoot(document.getElementById('app')!).render(
        <React.StrictMode>
            <App />
        </React.StrictMode>
    )
})
