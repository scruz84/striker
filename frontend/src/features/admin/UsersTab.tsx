import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import {
  useAdminUsers,
  useMe,
  useSetUserAdmin,
} from '../../api/hooks'

import { useConfirm } from '../../components/confirmDialogContext'
import { TeamNameLink } from '../../components/TeamNameLink'

export function UsersTab() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: users, isLoading } = useAdminUsers(true)

  return (
    <>
      <Typography variant="h6" sx={{ mb: 1 }}>
        {t('admin.usersTitle')}
      </Typography>
      {isLoading || !users || !loggedUser ? (
        <Skeleton variant="rectangular" height={200} />
      ) : (
        <UsersTable users={users} loggedUserId={Number(loggedUser.id)} />
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

  const handleToggleAdmin = async (user: (typeof users)[number]) => {
    const makeAdmin = !user.isAdmin
    const ok = await confirm({
      message: t(makeAdmin ? 'admin.usersTable.confirmGrantAdmin' : 'admin.usersTable.confirmRevokeAdmin', { name: user.name }),
      confirmLabel: t(makeAdmin ? 'admin.usersTable.grantAdmin' : 'admin.usersTable.revokeAdmin'),
      destructive: !makeAdmin,
    })
    if (ok) {
      setUserAdmin.mutate({ userId: user.userId, isAdmin: makeAdmin })
    }
  }

  return (
    <TableContainer component={Paper}>
      <Table size="small" aria-label="Users">
        <TableHead>
          <TableRow>
            <TableCell>{t('admin.usersTable.id')}</TableCell>
            <TableCell>{t('admin.usersTable.name')}</TableCell>
            <TableCell>{t('admin.usersTable.email')}</TableCell>
            <TableCell>{t('admin.usersTable.country')}</TableCell>
            <TableCell>{t('admin.usersTable.team')}</TableCell>
            <TableCell>{t('admin.usersTable.admin')}</TableCell>
            <TableCell align="right" />
          </TableRow>
        </TableHead>
        <TableBody>
          {users.map((user) => (
            <TableRow key={user.userId}>
              <TableCell>{user.userId}</TableCell>
              <TableCell>{user.name}</TableCell>
              <TableCell>{user.email}</TableCell>
              <TableCell>{user.country}</TableCell>
              <TableCell>{user.teamId !== null ? <TeamNameLink teamId={user.teamId}>{user.teamName}</TeamNameLink> : '-'}</TableCell>
              <TableCell>{user.isAdmin && <Chip size="small" color="primary" label={t('admin.usersTable.admin')} />}</TableCell>
              <TableCell align="right">
                <Button
                  size="small"
                  color={user.isAdmin ? 'secondary' : 'primary'}
                  onClick={() => handleToggleAdmin(user)}
                  disabled={setUserAdmin.isPending || user.userId === loggedUserId}
                >
                  {user.isAdmin ? t('admin.usersTable.revokeAdmin') : t('admin.usersTable.grantAdmin')}
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}
