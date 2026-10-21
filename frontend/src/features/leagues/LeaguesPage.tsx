import { useState } from 'react'
import EmojiEventsTwoToneIcon from '@mui/icons-material/EmojiEventsTwoTone'
import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import Typography from '@mui/material/Typography'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useLeagueDirectory } from '../../api/hooks'
import { COUNTRY_NAME_KEYS, type CountryLeagueDirectory, type LeagueDirectoryEntry } from '../../api/types'
import { PageHeader } from '../../components/PageHeader'

/** Lets any signed-in manager browse every league in the game world - not just their own -
 * and drop into a division's standings and roster (DivisionDetailsPage, unchanged) from there.
 * Only public information is shown: standings and squads, never another team's finances or
 * contract terms - see backend's Player.RedactFinancials. */
export function LeaguesPage() {
  const { t } = useTranslation()
  const { data: directory, isLoading } = useLeagueDirectory()
  const [countryIndex, setCountryIndex] = useState(0)

  if (isLoading || !directory) {
    return <Skeleton variant="rectangular" height={300} sx={{ borderRadius: 2 }} />
  }

  if (directory.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary">
        {t('leagues.none')}
      </Typography>
    )
  }

  const selected = directory[Math.min(countryIndex, directory.length - 1)]

  return (
    <Stack spacing={2}>
      <PageHeader title={t('leagues.title')} subtitle={t('leagues.subtitle')} />

      <Paper variant="outlined">
        <Tabs
          value={countryIndex}
          onChange={(_e, value) => setCountryIndex(value)}
          variant="scrollable"
          scrollButtons="auto"
          sx={{ borderBottom: 1, borderColor: 'divider' }}
        >
          {directory.map((entry) => (
            <Tab key={entry.country} label={t(COUNTRY_NAME_KEYS[entry.country] ?? entry.country)} />
          ))}
        </Tabs>
        <Box sx={{ p: 2 }}>
          <CountryTiers country={selected} />
        </Box>
      </Paper>
    </Stack>
  )
}

function CountryTiers({ country }: { country: CountryLeagueDirectory }) {
  const { t } = useTranslation()
  const tiers = new Map<number, LeagueDirectoryEntry[]>()
  for (const league of country.leagues) {
    const group = tiers.get(league.tier) ?? []
    group.push(league)
    tiers.set(league.tier, group)
  }
  const sortedTiers = Array.from(tiers.keys()).sort((a, b) => a - b)

  return (
    <Stack spacing={2}>
      {sortedTiers.map((tier) => {
        const leagues = [...(tiers.get(tier) ?? [])].sort((a, b) => a.groupNumber - b.groupNumber)
        return (
          <Box key={tier}>
            <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 1 }}>
              {tier === 1 && <EmojiEventsTwoToneIcon color="primary" fontSize="small" />}
              <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                {t('leagues.tier', { tier })}
              </Typography>
            </Stack>
            <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap', gap: 1 }}>
              {leagues.map((league) => (
                <Chip
                  key={league.leagueId}
                  label={league.name}
                  component={RouterLink}
                  to={`/division/${league.leagueId}`}
                  clickable
                  variant="outlined"
                />
              ))}
            </Stack>
          </Box>
        )
      })}
    </Stack>
  )
}
