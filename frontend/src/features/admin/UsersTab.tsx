import { useState } from 'react'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import {
  useAdminDeleteUser,
  useAdminUsers,
  useMe,
  useSetUserAdmin,
} from '../../api/hooks'

import { useConfirm } from '../../components/confirmDialogContext'
import { SearchBar } from '../../components/m3/SearchBar'
import { TeamNameLink } from '../../components/TeamNameLink'
import { filterUsers } from './userFilter'
import { useAdminFeedback } from './useAdminFeedback'
import { useIsPhone } from './useIsPhone'

// The id and country columns are the least useful ones; they only show once there is room for them
// (the full table needs a wide screen - below `md` the list stacks as cards instead).
const secondaryColumn = { display: { xs: 'none', lg: 'table-cell' } }

export function UsersTab() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: users, isLoading } = useAdminUsers(true)
  const [query, setQuery] = useState('')

  if (isLoading || !users || !loggedUser) {
    return <Skeleton variant="rectangular" height={200} />
  }

  const shown = filterUsers(users, query)

  return (
    <>
      <Typography variant="h6" sx={{ mb: 1 }}>
        {t('admin.usersTitle')}
      </Typography>
      <SearchBar
        compact
        value={query}
        onChange={setQuery}
        placeholder={t('admin.usersSearch.placeholder')}
        sx={{ mb: 2, maxWidth: { sm: 420 } }}
      />
      {shown.length === 0 ? (
        <Typography color="text.secondary">{t('admin.usersSearch.none', { query: query.trim() })}</Typography>
      ) : (
        <UsersTable users={shown} loggedUserId={Number(loggedUser.id)} />
      )}
    </>
  )
}

function UsersTable({
  users,
  loggedUserId,
}: {
  users: NonNullable<ReturnType<typeof useAdminUsers>['data']>
  loggedUserId: number
}) {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const setUserAdmin = useSetUserAdmin()
  const deleteUser = useAdminDeleteUser()
  const feedback = useAdminFeedback()
  const stackCards = useIsPhone('md')

  const handleToggleAdmin = async (user: (typeof users)[number]) => {
    const makeAdmin = !user.isAdmin
    const ok = await confirm({
      message: t(makeAdmin ? 'admin.usersTable.confirmGrantAdmin' : 'admin.usersTable.confirmRevokeAdmin', { name: user.name }),
      confirmLabel: t(makeAdmin ? 'admin.usersTable.grantAdmin' : 'admin.usersTable.revokeAdmin'),
      destructive: !makeAdmin,
    })
    if (ok) {
      setUserAdmin.mutate({ userId: user.userId, isAdmin: makeAdmin }, feedback(makeAdmin ? 'admin.usersTable.adminGranted' : 'admin.usersTable.adminRevoked'))
    }
  }

  const handleDelete = async (user: (typeof users)[number]) => {
    const ok = await confirm({
      message: t('admin.usersTable.confirmDelete', { name: user.name, email: user.email }),
      confirmLabel: t('admin.usersTable.delete'),
      destructive: true,
    })
    if (ok) {
      deleteUser.mutate(user.userId, feedback('admin.usersTable.deleted'))
    }
  }
  const lastSeen = (user: (typeof users)[number]) => {
    const days = Math.floor((Date.now() - new Date(user.lastSeenAt).getTime()) / 86_400_000)
    return days < 1 ? t('admin.usersTable.today') : t('admin.usersTable.lastSeenDays', { count: days })
  }
  const adminAction = (user: (typeof users)[number]) => (
    <Button
      size="small"
      color={user.isAdmin ? 'secondary' : 'primary'}
      onClick={() => handleToggleAdmin(user)}
      disabled={setUserAdmin.isPending || user.userId === loggedUserId}
    >
      {user.isAdmin ? t('admin.usersTable.revokeAdmin') : t('admin.usersTable.grantAdmin')}
    </Button>
  )
  // An admin cannot erase their own account here (they use the Account page like anyone else).
  const deleteAction = (user: (typeof users)[number]) => (
    <Button size="small" color="error" onClick={() => handleDelete(user)} disabled={deleteUser.isPending || user.userId === loggedUserId}>
      {t('admin.usersTable.delete')}
    </Button>
  )
  const teamLink = (user: (typeof users)[number]) =>
    user.teamId !== null ? <TeamNameLink teamId={user.teamId}>{user.teamName}</TeamNameLink> : '-'

  if (stackCards) {
    return (
      <Stack spacing={1.5}>
        {users.map((user) => (
          <Paper key={user.userId} variant="outlined" sx={{ p: 1.5 }}>
            <Stack direction="row" spacing={1} sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
              <Typography sx={{ fontWeight: 600, minWidth: 0, overflowWrap: 'anywhere' }}>{user.name}</Typography>
              {user.isAdmin && <Chip size="small" color="primary" label={t('admin.usersTable.admin')} />}
            </Stack>
            <Typography variant="body2" color="text.secondary" sx={{ overflowWrap: 'anywhere' }}>
              {user.email}
            </Typography>
            <Typography variant="body2" sx={{ mt: 0.5 }}>
              {user.country} · {teamLink(user)}
            </Typography>
            <Stack direction="row" sx={{ mt: 1, alignItems: 'center', justifyContent: 'space-between' }}>
              <Typography variant="caption" color="text.secondary">
                {t('admin.usersTable.id')} {user.userId}
              </Typography>
              <Stack direction="row" spacing={0.5}>
                {adminAction(user)}
                {deleteAction(user)}
              </Stack>
            </Stack>
            <Typography variant="caption" color="text.secondary" component="div">
              {t('admin.usersTable.lastSeen')}: {lastSeen(user)}
            </Typography>
          </Paper>
        ))}
      </Stack>
    )
  }

  return (
    <TableContainer component={Paper}>
      <Table size="small" aria-label="Users">
        <TableHead>
          <TableRow>
            <TableCell sx={secondaryColumn}>{t('admin.usersTable.id')}</TableCell>
            <TableCell>{t('admin.usersTable.name')}</TableCell>
            <TableCell>{t('admin.usersTable.email')}</TableCell>
            <TableCell sx={secondaryColumn}>{t('admin.usersTable.country')}</TableCell>
            <TableCell>{t('admin.usersTable.team')}</TableCell>
            <TableCell>{t('admin.usersTable.admin')}</TableCell>
            <TableCell>{t('admin.usersTable.lastSeen')}</TableCell>
            <TableCell align="right" />
          </TableRow>
        </TableHead>
        <TableBody>
          {users.map((user) => (
            <TableRow key={user.userId}>
              <TableCell sx={secondaryColumn}>{user.userId}</TableCell>
              <TableCell>{user.name}</TableCell>
              <TableCell>{user.email}</TableCell>
              <TableCell sx={secondaryColumn}>{user.country}</TableCell>
              <TableCell>{teamLink(user)}</TableCell>
              <TableCell>{user.isAdmin && <Chip size="small" color="primary" label={t('admin.usersTable.admin')} />}</TableCell>
              <TableCell>{lastSeen(user)}</TableCell>
              <TableCell align="right">
                {adminAction(user)}
                {deleteAction(user)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}
