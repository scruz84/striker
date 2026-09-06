import { createContext, useContext, type ReactNode } from 'react'

export interface ConfirmOptions {
  title?: string
  message: ReactNode
  confirmLabel?: string
  /** Renders the confirm button in the error color, for actions that are destructive or hard to undo. */
  destructive?: boolean
}

export type ConfirmFn = (options: ConfirmOptions) => Promise<boolean>

export const ConfirmContext = createContext<ConfirmFn | null>(null)

// Renders one shared dialog for the whole app (see ConfirmDialog.tsx's ConfirmProvider) rather
// than making every feature manage its own open/close state - callers just
// `await confirm({ message })` and get back whether the user accepted, right where they'd
// otherwise have called mutate() directly.
export function useConfirm(): ConfirmFn {
  const confirm = useContext(ConfirmContext)
  if (!confirm) {
    throw new Error('useConfirm must be used within a ConfirmProvider')
  }
  return confirm
}
