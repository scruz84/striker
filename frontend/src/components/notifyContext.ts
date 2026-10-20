import { createContext, useContext } from 'react'

export interface Notify {
  success: (message: string) => void
  error: (message: string) => void
}

export const NotifyContext = createContext<Notify | null>(null)

// A transient result message (a Snackbar rendered once for the whole app, see
// NotifyProvider.tsx) so a form can report "saved" / "failed" without every one of them leaving its
// own permanent Alert behind in the page. Validation errors tied to a specific field still belong
// in that field's own helper text.
export function useNotify(): Notify {
  const notify = useContext(NotifyContext)
  if (!notify) {
    throw new Error('useNotify must be used within a NotifyProvider')
  }
  return notify
}
