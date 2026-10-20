import { useMemo, useState, type ReactNode } from 'react'
import Alert from '@mui/material/Alert'
import Snackbar from '@mui/material/Snackbar'
import { NotifyContext, type Notify } from './notifyContext'

interface Message {
  key: number
  severity: 'success' | 'error'
  text: string
}

const SUCCESS_DURATION_MS = 4000
// An error is something the user has to read and act on, so it stays up a little longer.
const ERROR_DURATION_MS = 7000

export function NotifyProvider({ children }: { children: ReactNode }) {
  const [message, setMessage] = useState<Message | null>(null)
  const [open, setOpen] = useState(false)

  const notify = useMemo<Notify>(() => {
    const show = (severity: Message['severity']) => (text: string) => {
      // A newer message replaces the one on screen rather than queueing behind it: the latest
      // result is the one that matters.
      setMessage({ key: Date.now() + Math.random(), severity, text })
      setOpen(true)
    }
    return { success: show('success'), error: show('error') }
  }, [])

  return (
    <NotifyContext.Provider value={notify}>
      {children}
      <Snackbar
        key={message?.key}
        open={open}
        autoHideDuration={message?.severity === 'error' ? ERROR_DURATION_MS : SUCCESS_DURATION_MS}
        onClose={(_event, reason) => {
          if (reason !== 'clickaway') setOpen(false)
        }}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert severity={message?.severity ?? 'success'} variant="filled" onClose={() => setOpen(false)} sx={{ width: '100%' }}>
          {message?.text}
        </Alert>
      </Snackbar>
    </NotifyContext.Provider>
  )
}
