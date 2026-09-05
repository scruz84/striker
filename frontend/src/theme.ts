import { createTheme } from '@mui/material/styles'
import { green } from '@mui/material/colors'

export const lightTheme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: green[500],
    },
  },
})

export const darkTheme = createTheme({
  palette: {
    mode: 'dark',
    primary: {
      main: green[400],
    },
    background: {
      default: '#121212',
      paper: '#1e1e1e',
    },
  },
})
