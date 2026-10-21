import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import '@fontsource/oswald/600.css'
import '@fontsource/oswald/700.css'
import '@fontsource/roboto/300.css'
import '@fontsource/roboto/400.css'
import '@fontsource/roboto/500.css'
import '@fontsource/roboto/700.css'
import './i18n'
import './index.css'
import App from './app/App.tsx'
import { recoverFromStaleDeploy } from './recoverFromStaleDeploy'
import { registerServiceWorker } from './registerServiceWorker'

recoverFromStaleDeploy()
registerServiceWorker()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
