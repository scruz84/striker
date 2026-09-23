import { useState, type ReactNode } from 'react'
import TablePagination from '@mui/material/TablePagination'
import { useTranslation } from 'react-i18next'

const ROWS_PER_PAGE_OPTIONS = [25, 50, 100]

interface PaginatedProps<T> {
  rows: T[]
  // Any change to this key (e.g. a serialized filter/sort state) sends the list back to its first
  // page, so narrowing a search never leaves the reader stranded on a page that no longer exists.
  resetKey?: string
  children: (pageRows: T[]) => ReactNode
}

// Paginated renders only one page of an already filtered/sorted list plus a pager underneath -
// the market's listing endpoints return every row at once (there can be thousands of listed
// players or available coaches), and mounting a table row per item made those tabs sluggish.
// The page lives here rather than in each tab so it can be dropped in below their early returns.
export function Paginated<T>({ rows, resetKey = '', children }: PaginatedProps<T>) {
  const { t } = useTranslation()
  const [rowsPerPage, setRowsPerPage] = useState(ROWS_PER_PAGE_OPTIONS[0])
  const [pageState, setPageState] = useState({ page: 0, resetKey })

  const lastPage = Math.max(0, Math.ceil(rows.length / rowsPerPage) - 1)
  // Clamped too, so a list shrinking underneath (e.g. a listing bought or withdrawn) can't leave
  // an empty page showing.
  const page = pageState.resetKey === resetKey ? Math.min(pageState.page, lastPage) : 0
  const pageRows = rows.slice(page * rowsPerPage, (page + 1) * rowsPerPage)

  return (
    <>
      {children(pageRows)}
      {rows.length > ROWS_PER_PAGE_OPTIONS[0] && (
        <TablePagination
          component="div"
          count={rows.length}
          page={page}
          rowsPerPage={rowsPerPage}
          rowsPerPageOptions={ROWS_PER_PAGE_OPTIONS}
          onPageChange={(_event, newPage) => setPageState({ page: newPage, resetKey })}
          onRowsPerPageChange={(event) => {
            setRowsPerPage(Number(event.target.value))
            setPageState({ page: 0, resetKey })
          }}
          labelRowsPerPage={t('common.pagination.rowsPerPage')}
          labelDisplayedRows={({ from, to, count }) => t('common.pagination.displayedRows', { from, to, count })}
          sx={{ borderTop: 1, borderColor: 'divider' }}
        />
      )}
    </>
  )
}
