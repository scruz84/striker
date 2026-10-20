import { useTheme } from '@mui/material/styles'
import useMediaQuery from '@mui/material/useMediaQuery'

// True below the given breakpoint (`sm` by default: a phone held upright). The admin tables switch
// to stacked cards there instead of scrolling sideways, and its dialogs open full screen. A table
// with many columns (the users list) passes 'md' so it also stacks on a tablet, where its last
// column - the action button - would otherwise sit out of view.
export function useIsPhone(below: 'sm' | 'md' = 'sm'): boolean {
  const theme = useTheme()
  return useMediaQuery(theme.breakpoints.down(below))
}
