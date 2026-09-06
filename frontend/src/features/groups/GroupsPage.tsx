import { useState, type FormEvent } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import Grid from '@mui/material/Grid'
import List from '@mui/material/List'
import ListItemButton from '@mui/material/ListItemButton'
import ListItemText from '@mui/material/ListItemText'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import {
  useCreateGroup,
  useDeleteGroup,
  useGroupDetails,
  useJoinGroup,
  useLeaveGroup,
  useMe,
  useMyGroups,
} from '../../api/hooks'
import { useConfirm } from '../../components/confirmDialogContext'
import { formatCurrency } from '../../utils/formatCurrency'

export function GroupsPage() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: groups, isLoading } = useMyGroups(!!loggedUser)
  const [selectedGroupId, setSelectedGroupId] = useState<number | null>(null)

  const createGroup = useCreateGroup()
  const joinGroup = useJoinGroup()
  const [name, setName] = useState('')
  const [code, setCode] = useState('')

  const handleCreate = (event: FormEvent) => {
    event.preventDefault()
    if (!name.trim()) return
    createGroup.mutate(name.trim(), { onSuccess: () => setName('') })
  }

  const handleJoin = (event: FormEvent) => {
    event.preventDefault()
    if (!code.trim()) return
    joinGroup.mutate(code.trim().toUpperCase(), { onSuccess: () => setCode('') })
  }

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        {t('groups.title')}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        {t('groups.subtitle')}
      </Typography>

      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, sm: 6 }}>
          <Paper sx={{ p: 2 }} component="form" onSubmit={handleCreate}>
            <Typography variant="subtitle1" gutterBottom>
              {t('groups.create.title')}
            </Typography>
            <Stack direction="row" spacing={1}>
              <TextField size="small" label={t('groups.create.name')} value={name} onChange={(e) => setName(e.target.value)} fullWidth />
              <Button type="submit" variant="contained" disabled={createGroup.isPending}>
                {t('groups.create.button')}
              </Button>
            </Stack>
          </Paper>
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>
          <Paper sx={{ p: 2 }} component="form" onSubmit={handleJoin}>
            <Typography variant="subtitle1" gutterBottom>
              {t('groups.join.title')}
            </Typography>
            <Stack direction="row" spacing={1}>
              <TextField size="small" label={t('groups.join.code')} value={code} onChange={(e) => setCode(e.target.value)} fullWidth />
              <Button type="submit" variant="outlined" disabled={joinGroup.isPending}>
                {t('groups.join.button')}
              </Button>
            </Stack>
            {joinGroup.isError && (
              <Alert severity="error" sx={{ mt: 1 }}>
                {t('groups.join.error')}
              </Alert>
            )}
          </Paper>
        </Grid>
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, sm: 4 }}>
          <Paper>
            <Typography variant="subtitle1" sx={{ p: 2, pb: 0 }}>
              {t('groups.yourGroups')}
            </Typography>
            {isLoading || !groups ? (
              <Skeleton variant="rectangular" height={100} />
            ) : groups.length === 0 ? (
              <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
                {t('groups.noGroups')}
              </Typography>
            ) : (
              <List>
                {groups.map((group) => (
                  <ListItemButton
                    key={group.groupId}
                    selected={selectedGroupId === group.groupId}
                    onClick={() => setSelectedGroupId(group.groupId)}
                  >
                    <ListItemText primary={group.name} secondary={t('groups.code', { code: group.inviteCode })} />
                  </ListItemButton>
                ))}
              </List>
            )}
          </Paper>
        </Grid>
        <Grid size={{ xs: 12, sm: 8 }}>
          {selectedGroupId && <GroupLeaderboard groupId={selectedGroupId} onLeave={() => setSelectedGroupId(null)} />}
        </Grid>
      </Grid>
    </Box>
  )
}

function GroupLeaderboard({ groupId, onLeave }: { groupId: number; onLeave: () => void }) {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const { data: loggedUser } = useMe()
  const { data, isLoading } = useGroupDetails(String(groupId), true)
  const leaveGroup = useLeaveGroup()
  const deleteGroup = useDeleteGroup()

  if (isLoading || !data) {
    return <Skeleton variant="rectangular" height={200} />
  }

  const isOwner = loggedUser && data.group.ownerId === Number(loggedUser.id)

  const handleLeave = async () => {
    const ok = await confirm({
      message: t('groups.confirmLeave', { name: data.group.name }),
      confirmLabel: t('groups.leave'),
      destructive: true,
    })
    if (ok) {
      leaveGroup.mutate(groupId, { onSuccess: onLeave })
    }
  }

  const handleDelete = async () => {
    const ok = await confirm({
      message: t('groups.confirmDelete', { name: data.group.name }),
      confirmLabel: t('groups.deleteGroup'),
      destructive: true,
    })
    if (ok) {
      deleteGroup.mutate(groupId, { onSuccess: onLeave })
    }
  }

  return (
    <Paper>
      <Stack direction="row" sx={{ p: 2, justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="subtitle1">{data.group.name}</Typography>
        <Stack direction="row" spacing={1}>
          <Button size="small" color="secondary" onClick={handleLeave} disabled={leaveGroup.isPending}>
            {t('groups.leave')}
          </Button>
          {isOwner && (
            <Button size="small" color="error" onClick={handleDelete} disabled={deleteGroup.isPending}>
              {t('groups.deleteGroup')}
            </Button>
          )}
        </Stack>
      </Stack>
      <TableContainer>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>{t('groups.table.manager')}</TableCell>
              <TableCell>{t('groups.table.team')}</TableCell>
              <TableCell>{t('groups.table.league')}</TableCell>
              <TableCell align="right">{t('groups.table.budget')}</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {data.leaderboard.map((member) => (
              <TableRow key={member.userId}>
                <TableCell>
                  {member.userName}
                  {loggedUser && member.userId === Number(loggedUser.id) && (
                    <Chip size="small" label={t('groups.you')} sx={{ ml: 1 }} />
                  )}
                </TableCell>
                <TableCell>{member.teamName || '-'}</TableCell>
                <TableCell>{member.leagueName || '-'}</TableCell>
                <TableCell align="right">{member.budget !== null ? formatCurrency(member.budget) : '-'}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  )
}
