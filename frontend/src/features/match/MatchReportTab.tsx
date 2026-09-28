import type { WeekMatch } from '../../api/types'
import { MatchEventsReport } from './MatchEventsReport'

// A finished match's narrative report - just MatchEventsReport fed from a full WeekMatch (weather
// flavor plus the "full time" closing line, both only available/meaningful once the match is
// actually over). See MatchEventsReport for the shared live/finished rendering.
export function MatchReportTab({ match }: { match: WeekMatch }) {
  return (
    <MatchEventsReport
      events={match.events ?? []}
      homeId={match.home}
      homeName={match.homeName}
      awayId={match.away}
      awayName={match.awayName}
      weather={match.weather}
      attendance={match.attendance}
      refereeName={match.refereeName}
      assistant1Name={match.assistant1Name}
      assistant2Name={match.assistant2Name}
      final
    />
  )
}
