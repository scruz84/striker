import { useEffect, useMemo, useState, type ReactNode } from 'react'
import CssBaseline from '@mui/material/CssBaseline'
import { ThemeProvider } from '@mui/material/styles'
import { darkTheme, lightTheme } from '../theme'
import { getInitialMode, STORAGE_KEY, ThemeModeContext, type ThemeMode } from './themeMode'

export function ThemeModeProvider({ children }: { children: ReactNode }) {
  const [mode, setMode] = useState<ThemeMode>(getInitialMode)

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, mode)
    } catch {
      // Best-effort persistence only - the toggle still works for the rest of this session.
    }
  }, [mode])

  const value = useMemo(
    () => ({ mode, toggleMode: () => setMode((prev: ThemeMode) => (prev === 'light' ? 'dark' : 'light')) }),
    [mode],
  )

  return (
    <ThemeModeContext.Provider value={value}>
      <ThemeProvider theme={mode === 'dark' ? darkTheme : lightTheme}>
        <CssBaseline enableColorScheme />
        {children}
      </ThemeProvider>
    </ThemeModeContext.Provider>
  )
}
