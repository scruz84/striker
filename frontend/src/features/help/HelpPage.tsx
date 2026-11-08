import type { SvgIconComponent } from '@mui/icons-material'
import AccountBalanceWalletOutlinedIcon from '@mui/icons-material/AccountBalanceWalletOutlined'
import DashboardOutlinedIcon from '@mui/icons-material/DashboardOutlined'
import EmojiEventsOutlinedIcon from '@mui/icons-material/EmojiEventsOutlined'
import FlagOutlinedIcon from '@mui/icons-material/FlagOutlined'
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined'
import HandshakeOutlinedIcon from '@mui/icons-material/HandshakeOutlined'
import PublicOutlinedIcon from '@mui/icons-material/PublicOutlined'
import SchoolOutlinedIcon from '@mui/icons-material/SchoolOutlined'
import SmartphoneOutlinedIcon from '@mui/icons-material/SmartphoneOutlined'
import SportsSoccerOutlinedIcon from '@mui/icons-material/SportsSoccerOutlined'
import SwapHorizOutlinedIcon from '@mui/icons-material/SwapHorizOutlined'
import Accordion from '@mui/material/Accordion'
import AccordionDetails from '@mui/material/AccordionDetails'
import AccordionSummary from '@mui/material/AccordionSummary'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import Box from '@mui/material/Box'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { PageHeader } from '../../components/PageHeader'
import { LegalLinks } from '../legal/LegalLinks'

interface HelpItem {
  q: string
  a: string
}

const SECTIONS: { key: string; icon: SvgIconComponent }[] = [
  { key: 'access', icon: SmartphoneOutlinedIcon },
  { key: 'gettingStarted', icon: FlagOutlinedIcon },
  { key: 'dashboard', icon: DashboardOutlinedIcon },
  { key: 'lineup', icon: SportsSoccerOutlinedIcon },
  { key: 'matches', icon: SportsSoccerOutlinedIcon },
  { key: 'transfers', icon: SwapHorizOutlinedIcon },
  { key: 'squadDevelopment', icon: SchoolOutlinedIcon },
  { key: 'finances', icon: AccountBalanceWalletOutlinedIcon },
  { key: 'leagues', icon: PublicOutlinedIcon },
  { key: 'cup', icon: EmojiEventsOutlinedIcon },
  { key: 'friendlies', icon: HandshakeOutlinedIcon },
  { key: 'nationalTeams', icon: FlagOutlinedIcon },
  { key: 'groups', icon: GroupsOutlinedIcon },
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
        <PageHeader title={t('help.title')} subtitle={t('help.subtitle')} />
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
      <LegalLinks sx={{ pt: 1 }} />
    </Stack>
  )
}
