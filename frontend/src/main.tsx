import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'

// Auto-reload once when a new service worker takes control
if ('serviceWorker' in navigator) {
  let reloading = false
  navigator.serviceWorker.addEventListener('controllerchange', () => {
    if (reloading) return
    reloading = true
    window.location.reload()
  })

  // Force an update check on every page load instead of waiting for the browser cycle
  navigator.serviceWorker.ready.then((reg) => reg.update())
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
)
