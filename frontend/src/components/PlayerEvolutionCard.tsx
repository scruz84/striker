import Paper from '@mui/material/Paper'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { PlayerEvolution, SkillDelta } from '../api/types'

const SKILL_ROWS: { field: 'goalKeeper' | 'defense' | 'lateral' | 'pass' | 'midfield' | 'dribbling' | 'shot'; labelKey: string }[] = [
  { field: 'goalKeeper', labelKey: 'playerCard.skills.goalkeeper' },
  { field: 'defense', labelKey: 'playerCard.skills.defense' },
  { field: 'lateral', labelKey: 'playerCard.skills.lateral' },
  { field: 'pass', labelKey: 'playerCard.skills.pass' },
  { field: 'midfield', labelKey: 'playerCard.skills.midfield' },
  { field: 'dribbling', labelKey: 'playerCard.skills.dribbling' },
  { field: 'shot', labelKey: 'playerCard.skills.shot' },
]

/** A player's skill/age changes from their most recent season-end progression run - only the
 * skills that actually drifted (see engine.driftSkillsForPosition's independent per-skill
 * chance) are listed, since most seasons leave most skills untouched. Split from
 * PlayerEvolutionCard so the squad-wide latest-evolutions list can reuse just this body under its
 * own per-player heading, instead of a second nested title. */
export function PlayerEvolutionBody({ evolution }: { evolution: PlayerEvolution }) {
  const { t } = useTranslation()
  const changedSkills = SKILL_ROWS.filter(({ field }) => evolution[field].before !== evolution[field].after)

  return (
    <>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 1 }}>
        <Typography variant="body2" color="text.secondary" sx={{ flex: 1 }}>
          {t('evolution.age')}
        </Typography>
        <SkillValues delta={evolution.age} />
      </Stack>
      {changedSkills.length === 0 ? (
        <Typography variant="body2" color="text.secondary">
          {t('evolution.noChange')}
        </Typography>
      ) : (
        <Stack spacing={0.75}>
          {changedSkills.map(({ field, labelKey }) => (
            <Stack key={field} direction="row" spacing={1} sx={{ alignItems: 'center' }}>
              <Typography variant="body2" sx={{ flex: 1 }}>
                {t(labelKey)}
              </Typography>
              <SkillValues delta={evolution[field]} />
            </Stack>
          ))}
        </Stack>
      )}
    </>
  )
}

/** PlayerEvolutionBody wrapped with its own card and "Season N" title - the player details page's
 * standalone section. */
export function PlayerEvolutionCard({ evolution }: { evolution: PlayerEvolution }) {
  const { t } = useTranslation()
  return (
    <Paper variant="outlined" sx={{ p: 2 }}>
      <Typography variant="subtitle1" gutterBottom>
        {t('evolution.title', { season: evolution.season })}
      </Typography>
      <PlayerEvolutionBody evolution={evolution} />
    </Paper>
  )
}

function SkillValues({ delta }: { delta: SkillDelta }) {
  const up = delta.after > delta.before
  const down = delta.after < delta.before
  return (
    <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
      <Typography variant="body2" color="text.secondary">
        {delta.before}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        &rarr;
      </Typography>
      <Typography
        variant="body2"
        sx={{ fontWeight: 700, minWidth: 18, textAlign: 'right', color: up ? 'success.main' : down ? 'error.main' : 'text.primary' }}
      >
        {delta.after}
      </Typography>
    </Stack>
  )
}
