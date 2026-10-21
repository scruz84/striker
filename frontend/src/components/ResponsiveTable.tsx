import type { ReactNode } from 'react'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import ListItemButton from '@mui/material/ListItemButton'
import Paper from '@mui/material/Paper'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import useMediaQuery from '@mui/material/useMediaQuery'

export interface ResponsiveTableColumn<T> {
  id: string
  header: ReactNode
  cell: (row: T) => ReactNode
  align?: 'left' | 'right' | 'center'
}

interface ResponsiveTableProps<T> {
  rows: T[]
  getKey: (row: T) => string | number
  /** Desktop table columns. */
  columns: ResponsiveTableColumn<T>[]
  /** Phone layout: a two-line row - main text left, key figure right. */
  mobile: {
    primary: (row: T) => ReactNode
    secondary?: (row: T) => ReactNode
    end?: (row: T) => ReactNode
    leading?: (row: T) => ReactNode
  }
  onRowClick?: (row: T) => void
  ariaLabel?: string
  /** Breakpoint from which the real table is used (default `md`). */
  tableFrom?: 'sm' | 'md'
}

/** Table on desktop, list of two-line rows on a phone: the principal datum on the left and the
 * key figure on the right, so nothing important is hidden behind horizontal scrolling. */
export function ResponsiveTable<T>({ rows, getKey, columns, mobile, onRowClick, ariaLabel, tableFrom = 'md' }: ResponsiveTableProps<T>) {
  const theme = useTheme()
  const showTable = useMediaQuery(theme.breakpoints.up(tableFrom))

  if (showTable) {
    return (
      <TableContainer component={Paper} variant="outlined">
        <Table size="small" aria-label={ariaLabel}>
          <TableHead>
            <TableRow>
              {columns.map((column) => (
                <TableCell key={column.id} align={column.align}>
                  {column.header}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((row) => (
              <TableRow
                key={getKey(row)}
                hover={!!onRowClick}
                onClick={onRowClick ? () => onRowClick(row) : undefined}
                sx={onRowClick ? { cursor: 'pointer' } : undefined}
              >
                {columns.map((column) => (
                  <TableCell key={column.id} align={column.align}>
                    {column.cell(row)}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    )
  }

  const renderRow = (row: T) => (
    <>
      {mobile.leading?.(row)}
      <Box sx={{ minWidth: 0, flexGrow: 1 }}>
        <Typography variant="body2" component="div" sx={{ fontWeight: 500 }}>
          {mobile.primary(row)}
        </Typography>
        {mobile.secondary && (
          <Typography variant="body2" color="text.secondary" component="div">
            {mobile.secondary(row)}
          </Typography>
        )}
      </Box>
      {mobile.end && (
        <Typography variant="body2" component="div" sx={{ flexShrink: 0, fontWeight: 600, textAlign: 'right', fontVariantNumeric: 'tabular-nums' }}>
          {mobile.end(row)}
        </Typography>
      )}
    </>
  )

  return (
    <Paper variant="outlined">
      <List disablePadding aria-label={ariaLabel}>
        {rows.map((row, index) => {
          const divider = index < rows.length - 1
          return onRowClick ? (
            <ListItem key={getKey(row)} disablePadding divider={divider}>
              <ListItemButton onClick={() => onRowClick(row)} sx={{ gap: 1.5, alignItems: 'center' }}>
                {renderRow(row)}
              </ListItemButton>
            </ListItem>
          ) : (
            <ListItem key={getKey(row)} divider={divider} sx={{ gap: 1.5, alignItems: 'center' }}>
              {renderRow(row)}
            </ListItem>
          )
        })}
      </List>
    </Paper>
  )
}
