import { MatchEventsReport } from './MatchEventsReport'
import type { MatchView } from './matchViewModel'

// A finished match's narrative report - just MatchEventsReport fed from the shared match view
// (weather flavor plus the "full time" closing line, both only available/meaningful once the match is
// actually over). See MatchEventsReport for the shared live/finished rendering.
export function MatchReportTab({ match }: { match: MatchView }) {
  return (
    <MatchEventsReport
      events={match.events ?? []}
      homeId={match.homeId}
      homeName={match.homeName}
      awayId={match.awayId}
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
