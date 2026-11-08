import { useRef } from 'react'
import Box from '@mui/material/Box'
import IconButton from '@mui/material/IconButton'
import InputBase from '@mui/material/InputBase'
import type { SxProps, Theme } from '@mui/material/styles'
import CloseIcon from '@mui/icons-material/Close'
import SearchIcon from '@mui/icons-material/Search'
import { useTranslation } from 'react-i18next'

interface SearchBarProps {
  value: string
  onChange: (value: string) => void
  /** Also the field's accessible name. */
  placeholder: string
  /** 48px instead of 56px, for a search field inside a row of filters. */
  compact?: boolean
  autoFocus?: boolean
  sx?: SxProps<Theme>
}

/** M3 search bar: a pill on surfaceContainerHigh with a leading search icon and, while it holds
 * text, a clear button that empties it and gives the focus back to the field (see the ui-theme
 * spec). Every free-text search field in the app uses it. */
export function SearchBar({ value, onChange, placeholder, compact = false, autoFocus, sx }: SearchBarProps) {
  const { t } = useTranslation()
  const inputRef = useRef<HTMLInputElement>(null)

  return (
    <Box
      sx={[
        {
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          height: compact ? 48 : 56,
          pl: 2,
          pr: 0.5,
          borderRadius: '9999px',
          bgcolor: 'surfaceContainerHigh',
          color: 'onSurfaceVariant',
          '&:focus-within': { outline: 2, outlineColor: 'primary.main', outlineStyle: 'solid', outlineOffset: -2 },
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      <SearchIcon />
      <InputBase
        inputRef={inputRef}
        type="search"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        autoFocus={autoFocus}
        inputProps={{ 'aria-label': placeholder }}
        sx={{
          flex: 1,
          minWidth: 0,
          alignSelf: 'stretch',
          color: 'onSurface',
          // The whole pill height is the tap target, not just the text line.
          '& input': { height: '100%', py: 0, boxSizing: 'border-box' },
          // The browser's own clear control would duplicate ours.
          '& input::-webkit-search-cancel-button': { display: 'none' },
        }}
      />
      {value !== '' && (
        <IconButton
          aria-label={t('common.clearSearch')}
          onClick={() => {
            onChange('')
            inputRef.current?.focus()
          }}
        >
          <CloseIcon />
        </IconButton>
      )}
    </Box>
  )
}
