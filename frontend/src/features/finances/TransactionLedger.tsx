import { useMemo, useState } from 'react'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import Collapse from '@mui/material/Collapse'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import ListItemButton from '@mui/material/ListItemButton'
import ListItemText from '@mui/material/ListItemText'
import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import ExpandLessIcon from '@mui/icons-material/ExpandLess'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import { useTranslation } from 'react-i18next'
import { TRANSACTION_REASON_KEYS, type TeamTransaction } from '../../api/types'
import { EmptyState } from '../../components/EmptyState'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDate, formatShortDate } from '../../utils/formatDate'
import { translateLedgerDescription } from '../../utils/ledgerDescription'

const GROUPS_PER_PAGE = 5

type LedgerFilter = 'all' | 'income' | 'expense'

export interface LedgerGroup {
  key: string
  date: string
  transactions: TeamTransaction[]
  income: number
  expenses: number
  net: number
}

/** Groups a most-recent-first list of transactions by calendar day (a matchday's wages,
 * maintenance, sponsorship and tickets land together), with per-day subtotals. */
export function groupTransactionsByDay(transactions: TeamTransaction[]): LedgerGroup[] {
  const groups: LedgerGroup[] = []
  const byKey = new Map<string, LedgerGroup>()
  for (const tx of transactions) {
    const day = new Date(tx.createdAt)
    const key = Number.isNaN(day.getTime()) ? tx.createdAt : `${day.getFullYear()}-${day.getMonth()}-${day.getDate()}`
    let group = byKey.get(key)
    if (!group) {
      group = { key, date: tx.createdAt, transactions: [], income: 0, expenses: 0, net: 0 }
      byKey.set(key, group)
      groups.push(group)
    }
    group.transactions.push(tx)
    if (tx.amount >= 0) {
      group.income += tx.amount
    } else {
      group.expenses += tx.amount
    }
    group.net += tx.amount
  }
  return groups
}

/** The account ledger: movements grouped by day with subtotals, collapsible, filterable by
 * income/expense chips and paginated with "load more" - instead of one 26-row wall of text. */
export function TransactionLedger({ transactions }: { transactions: TeamTransaction[] }) {
  const { t } = useTranslation()
  const [filter, setFilter] = useState<LedgerFilter>('all')
  const [visibleGroups, setVisibleGroups] = useState(GROUPS_PER_PAGE)
  const [toggled, setToggled] = useState<Record<string, boolean>>({})

  const groups = useMemo(() => {
    const filtered = transactions.filter((tx) => (filter === 'income' ? tx.amount >= 0 : filter === 'expense' ? tx.amount < 0 : true))
    return groupTransactionsByDay(filtered)
  }, [transactions, filter])

  if (transactions.length === 0) {
    return (
      <Paper variant="outlined">
        <EmptyState compact title={t('finances.noTransactions')} />
      </Paper>
    )
  }

  const shown = groups.slice(0, visibleGroups)
  // The newest day starts open, the rest folded; tapping a header flips its default.
  const isOpen = (key: string, index: number) => toggled[key] ?? index === 0

  return (
    <Box>
      <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 600, mb: 1 }}>
        {t('finances.ledger.title')}
      </Typography>
      <Stack direction="row" spacing={1} sx={{ mb: 1.5 }}>
        {(['all', 'income', 'expense'] as const).map((value) => (
          <Chip
            key={value}
            label={t(`finances.ledger.filter.${value}`)}
            color={filter === value ? 'primary' : 'default'}
            variant={filter === value ? 'filled' : 'outlined'}
            onClick={() => {
              setFilter(value)
              setVisibleGroups(GROUPS_PER_PAGE)
            }}
          />
        ))}
      </Stack>
      <Paper variant="outlined">
        <List disablePadding aria-label={t('finances.ledger.title')}>
          {shown.map((group, index) => (
            <Box key={group.key}>
              <ListItemButton
                onClick={() => setToggled((current) => ({ ...current, [group.key]: !isOpen(group.key, index) }))}
                aria-expanded={isOpen(group.key, index)}
                sx={{ bgcolor: 'action.hover', gap: 1 }}
              >
                <ListItemText
                  primary={formatDate(group.date)}
                  secondary={t('finances.ledger.subtotal', {
                    income: formatCurrency(group.income, { compact: true }),
                    expenses: formatCurrency(Math.abs(group.expenses), { compact: true }),
                  })}
                  slotProps={{ primary: { variant: 'subtitle2' }, secondary: { variant: 'caption' } }}
                />
                <Typography variant="subtitle2" sx={{ color: group.net < 0 ? 'error.main' : 'success.main', fontVariantNumeric: 'tabular-nums' }}>
                  {formatCurrency(group.net, { signed: true })}
                </Typography>
                {isOpen(group.key, index) ? <ExpandLessIcon /> : <ExpandMoreIcon />}
              </ListItemButton>
              <Collapse in={isOpen(group.key, index)} unmountOnExit>
                <List disablePadding>
                  {group.transactions.map((tx) => (
                    <ListItem key={tx.id} divider sx={{ gap: 1.5 }}>
                      <ListItemText
                        primary={t(TRANSACTION_REASON_KEYS[tx.reason] ?? 'finances.reasons.other')}
                        secondary={`${formatShortDate(tx.createdAt)} · ${translateLedgerDescription(t, tx.description)}`}
                        slotProps={{ primary: { variant: 'body2' }, secondary: { variant: 'caption' } }}
                      />
                      <Typography
                        variant="body2"
                        sx={{ flexShrink: 0, fontWeight: 600, fontVariantNumeric: 'tabular-nums', color: tx.amount < 0 ? 'error.main' : 'success.main' }}
                      >
                        {formatCurrency(tx.amount, { signed: true })}
                      </Typography>
                    </ListItem>
                  ))}
                </List>
              </Collapse>
            </Box>
          ))}
        </List>
        {groups.length > shown.length && (
          <Box sx={{ p: 1, textAlign: 'center' }}>
            <Button onClick={() => setVisibleGroups((count) => count + GROUPS_PER_PAGE)}>{t('finances.ledger.loadMore')}</Button>
          </Box>
        )}
      </Paper>
    </Box>
  )
}
