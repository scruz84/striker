import type { SvgIconComponent } from '@mui/icons-material'
import AccountBalanceWalletTwoToneIcon from '@mui/icons-material/AccountBalanceWalletTwoTone'
import DashboardTwoToneIcon from '@mui/icons-material/DashboardTwoTone'
import EmojiEventsTwoToneIcon from '@mui/icons-material/EmojiEventsTwoTone'
import FlagTwoToneIcon from '@mui/icons-material/FlagTwoTone'
import GroupsTwoToneIcon from '@mui/icons-material/GroupsTwoTone'
import HandshakeTwoToneIcon from '@mui/icons-material/HandshakeTwoTone'
import PublicTwoToneIcon from '@mui/icons-material/PublicTwoTone'
import SchoolTwoToneIcon from '@mui/icons-material/SchoolTwoTone'
import SportsSoccerTwoToneIcon from '@mui/icons-material/SportsSoccerTwoTone'
import SwapHorizTwoToneIcon from '@mui/icons-material/SwapHorizTwoTone'
import Accordion from '@mui/material/Accordion'
import AccordionDetails from '@mui/material/AccordionDetails'
import AccordionSummary from '@mui/material/AccordionSummary'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import Box from '@mui/material/Box'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'

interface HelpItem {
  q: string
  a: string
}

const SECTIONS: { key: string; icon: SvgIconComponent }[] = [
  { key: 'gettingStarted', icon: FlagTwoToneIcon },
  { key: 'dashboard', icon: DashboardTwoToneIcon },
  { key: 'lineup', icon: SportsSoccerTwoToneIcon },
  { key: 'matches', icon: SportsSoccerTwoToneIcon },
  { key: 'transfers', icon: SwapHorizTwoToneIcon },
  { key: 'squadDevelopment', icon: SchoolTwoToneIcon },
  { key: 'finances', icon: AccountBalanceWalletTwoToneIcon },
  { key: 'leagues', icon: PublicTwoToneIcon },
  { key: 'cup', icon: EmojiEventsTwoToneIcon },
  { key: 'friendlies', icon: HandshakeTwoToneIcon },
  { key: 'nationalTeams', icon: FlagTwoToneIcon },
  { key: 'groups', icon: GroupsTwoToneIcon },
]

type Translate = (key: string, options: { returnObjects: true }) => unknown

/** A static reference page explaining every game mechanic and option in the app - grouped into
 * the same sections as the site nav, each an expandable Q&A list pulled from help.sections.*.items
 * in the locale files. Purely informational: no data fetching, so it works whether or not the
 * visitor is signed in. */
export function HelpPage() {
  const { t } = useTranslation()

  return (
    <Stack spacing={2}>
      <Box>
        <Typography variant="h5" sx={{ fontWeight: 700 }}>
          {t('help.title')}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {t('help.subtitle')}
        </Typography>
      </Box>

      {SECTIONS.map(({ key, icon: Icon }) => {
        const items = (t as Translate)(`help.sections.${key}.items`, { returnObjects: true })
        if (!Array.isArray(items)) {
          return null
        }

        return (
          <Accordion key={key} variant="outlined" disableGutters>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center' }}>
                <Icon color="primary" fontSize="small" />
                <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                  {t(`help.sections.${key}.title`)}
                </Typography>
              </Stack>
            </AccordionSummary>
            <AccordionDetails>
              <Stack spacing={2}>
                {(items as HelpItem[]).map((item, index) => (
                  <Box key={index}>
                    <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                      {item.q}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      {item.a}
                    </Typography>
                  </Box>
                ))}
              </Stack>
            </AccordionDetails>
          </Accordion>
        )
      })}
    </Stack>
  )
}
