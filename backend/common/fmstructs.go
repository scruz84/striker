package common

import (
	"math"
	"time"
)

type ClassificationPosition struct {
	Position      int    `json:"position"`
	TeamId        int64  `json:"teamId"`
	TeamName      string `json:"teamName"`
	Points        int    `json:"points"`
	Matches       int    `json:"matches"`
	Wins          int    `json:"wins"`
	Draws         int    `json:"draws"`
	Losses        int    `json:"losses"`
	GoalsScored   int    `json:"goalsScored"`
	GoalsReceived int    `json:"goalsReceived"`
	// Seed is the team's place in the season's starting order (see engine.ComputeSeasonSeeds), the
	// last tie-break before the team id; nil for a team with no starting place.
	Seed *int `json:"seed"`
}

type Classification struct {
	Positions []ClassificationPosition `json:"positions"`
}

// TeamStanding is a team's rank-relevant classification data, used by the promotion/
// relegation algorithm (see engine.ComputePromotionRelegation). Not JSON-exposed.
type TeamStanding struct {
	TeamId      int64
	Points      int
	GoalDiff    int
	GoalsScored int
}

// LeagueTierInfo identifies which country and tier a league sits at in its promotion/
// relegation pyramid (1 = top division). Not JSON-exposed.
type LeagueTierInfo struct {
	LeagueId int64
	Country  string
	Tier     int
}

type DivisionDetails struct {
	Id                  int64          `json:"id"`
	Season              int            `json:"season"`
	Name                string         `json:"name"`
	Classification      Classification `json:"classification"`
	PreviousWeekMatches []WeekMatch    `json:"previousWeekMatches"`
	NextWeekMatches     []WeekMatch    `json:"nextWeekMatches"`
	// Zones is how many of this league's places sit in each promotion/relegation zone of its
	// country's pyramid (see PyramidZones) - all zero for an archived league or a single tier. They
	// let the classification table colour its rows.
	Zones Zones `json:"zones"`
	// Playoff is the current season's promotion playoff ties with a side in this league (see
	// GetLeaguePlayoffTies) - empty before the league ends and once the season has rolled over.
	Playoff []CupTie `json:"playoff"`
}

type LoginRequest struct {
	Provider string `json:"provider"`
	Jwt      string `json:"jwt"`
	// DevEmail selects a common.DevUsers entry to sign in as, bypassing Google entirely - only
	// honored when common.DevLoginEnabled() is true. See handlers.DoDevLogin.
	DevEmail string `json:"devEmail"`
}

type LoggedUser struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Provider   string `json:"provider"`
	Email      string `json:"email"`
	Country    string `json:"country"`
	TeamId     string `json:"teamId"`
	TeamName   string `json:"teamName"`
	TeamLeague string `json:"teamLeague"`
	PictureUrl string `json:"pictureUrl"`
	IsAdmin    bool   `json:"isAdmin"`
	// True for a signed-in user who hasn't picked a country yet (TeamId == "0") - the frontend
	// gates every other route behind the onboarding country picker while this is true. See
	// handlers.SelectCountryHandler.
	PendingCountrySelection bool `json:"pendingCountrySelection"`
	// TermsAccepted is false until the user accepted the current Privacy Policy and Terms (and
	// confirmed the minimum age) - the frontend gates every route behind /accept-terms until then.
	// TermsVersion is the current version they must accept (common.CurrentTermsVersion).
	TermsAccepted bool `json:"termsAccepted"`
	TermsVersion  int  `json:"termsVersion"`
	// ReleasedTeamName is set once, after their club was released for inactivity: onboarding tells
	// them, and it is cleared when they claim a new club.
	ReleasedTeamName string `json:"releasedTeamName,omitempty"`
}

type HttpResponse struct {
	Message string `json:"message"`
}

type Lineup struct {
	MatchId       int64                `json:"matchId"`
	TeamId        int64                `json:"teamId"`
	Formation     int                  `json:"formation"`
	Positions     []LineupPosition     `json:"positions"`
	Substitutions []LineupSubstitution `json:"substitutions"`
	// Mentality is how the side plays: MENTALITY_DEFENSIVE, _BALANCED (the default) or _ATTACKING
	// (see common/mentality.go).
	Mentality int `json:"mentality"`
	// Orders are the lineup's conditional match orders (see common/lineuporders.go). Like
	// ProjectedReplacements, only the lineup GET handlers a team owner opens fill it: a team's
	// orders are never shown to anyone else, only their effects (events) are.
	Orders []LineupOrder `json:"orders,omitempty"`
	// LineupRoles are the optional captain and set-piece takers (see common/lineuproles.go). The
	// takers are the owner's business: every lineup shown to other viewers goes through
	// RedactTakers first.
	LineupRoles
	// ProjectedReplacements is filled only by the lineup GET handlers a team owner opens (never
	// stored): for each unavailable starter, who the pre-kickoff refill would put in his slot if
	// the match kicked off now (see ProjectKickoffReplacements).
	ProjectedReplacements []ProjectedReplacement `json:"projectedReplacements,omitempty"`
}

type LineupPosition struct {
	PlayerId int64 `json:"playerId"`
	Position int   `json:"position"`
}

// CopyLineup returns an independent copy of lineup - specifically, one whose Positions slice can
// be mutated (e.g. by ApplySubstitutionToLineup) without affecting the original. Used wherever a
// caller needs to derive an evolving "lineup as it currently stands" from a fixed starting lineup
// without mutating the starting one out from under some other reader.
func CopyLineup(lineup *Lineup) *Lineup {
	if lineup == nil {
		return nil
	}
	cp := *lineup
	cp.Positions = append([]LineupPosition(nil), lineup.Positions...)
	return &cp
}

// ApplySubstitutionToLineup swaps a SUBSTITUTION event's outgoing player for its incoming one in
// lineup's own position list, in place, at whichever slot the outgoing player was holding.
//
// Nothing in the simulation pipeline updates a Lineup itself when a substitution happens -
// engine.applyScheduledSubstitutions/bringOnInjuryReplacement only swap the two teams' internal
// position-group maps the goal math actually reads (see engine/substitution_simulation.go), which
// is a different representation than the Lineup a live viewer's head-to-head pitch or a finished
// match's report renders (MatchHeadToHeadPitch, fed by WeekMatch.HomeLineup/AwayLineup or a live
// broadcast's own "lineups"/"event" messages) - so without this, both kept showing the XI a
// manager submitted before kickoff forever, never whoever's actually on the pitch after a
// substitution. Deriving the real on-pitch lineup means replaying every SUBSTITUTION event onto a
// copy (see CopyLineup) of the starting lineup after the fact - see engine.BroadcastMatchLive (a
// live broadcast, applied event by event as the broadcast paces through them) and WeekMatchById (a
// finished match's report, applied all at once from the full event history).
//
// No-op for anything but a SUBSTITUTION event, a nil lineup, an event without the expected
// [outgoingPlayerId, incomingPlayerId] pair, or an event for the other team's lineup (TeamId
// decides which of a match's two Lineups this event even applies to, so a caller can call this for
// both without checking which team an event belongs to first).
func ApplySubstitutionToLineup(lineup *Lineup, event Event) {
	if lineup == nil || event.EventType != SUBSTITUTION || len(event.TeamOwnerPlayers) != 2 || event.TeamOwner != lineup.TeamId {
		return
	}
	outId, inId := event.TeamOwnerPlayers[0], event.TeamOwnerPlayers[1]
	for i, pos := range lineup.Positions {
		if pos.PlayerId == outId {
			lineup.Positions[i].PlayerId = inId
			return
		}
	}
}

type Team struct {
	Id       int64  `json:"id"`
	OwnerId  int64  `json:"ownerId"`
	Country  string `json:"country"`
	TeamName string `json:"name"`
	// IsNationalTeam marks a country's unowned national team (see common.CreateNationalTeam) -
	// GetTeamDetailsById's one signal to the frontend that this team has no division, stadium, or
	// manager, and that its "squad" is the country's current pool of eligible players rather than
	// a real roster.
	IsNationalTeam         bool   `json:"isNationalTeam,omitempty"`
	DivisionId             int64  `json:"divisionId"`
	DivisionName           string `json:"divisionName"`
	ClassificationPosition int    `json:"classificationPosition"`
	// NameChanged is only populated by GetTeamByOwner (the one place the rename UI reads it
	// from) - other lookups leave it at the zero value, since nothing else consumes it.
	NameChanged bool `json:"nameChanged"`
	// AiLevel is an internal difficulty parameter engine.RunAITeamManagement reads to decide how
	// to manage an unowned team (see AiLevelCaretaker/AiLevelManager/AiLevelDirector) - not a fact
	// any manager, owned team or not, is meant to see (#19), so GetTeamById is the only thing that
	// ever populates it; GetTeamDetailsById (every manager-facing team lookup) clears it back to
	// 0 before it can reach a JSON response.
	AiLevel int `json:"aiLevel,omitempty"`
	// AiStyle is a CPU club's style (see common.AiClubStyle*), read only by GetTeamById for the CPU
	// routines - never sent to clients.
	AiStyle string `json:"-"`
	// DissolvedSeason is the last season a dissolved club played (see DissolveTeam), 0 for every
	// club still playing. The team page shows "Dissolved in season N" and no squad or actions.
	DissolvedSeason int `json:"dissolvedSeason,omitempty"`
}

type Player struct {
	Id int64 `json:"id"`
	// TeamId is 0 for a free agent (team_id IS NULL in the database - see GetFreeAgents):
	// GetPlayerById coalesces the NULL to 0 rather than requiring every caller to know whether
	// the player it looked up happens to be currently unowned.
	TeamId    int64  `json:"teamId"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	// BirthDay is the anchor for this player's periodic in-game birthday, not a real
	// date-of-birth: it's set to the wall-clock creation time when the player is generated, and
	// rolled forward by engine.BirthdayCycleDays every time engine.ProcessBirthdays fires their
	// next birthday (which is also what bumps Age - see that function). The player details page
	// derives "next birthday in N days" from this field directly.
	BirthDay   time.Time    `json:"birthDay"`
	GoalKeeper int          `json:"goalKeeper"`
	Defense    int          `json:"defense"`
	Lateral    int          `json:"lateral"`
	Pass       int          `json:"pass"`
	Midfield   int          `json:"midfield"`
	Dribbling  int          `json:"dribbling"`
	Shot       int          `json:"shot"`
	Header     bool         `json:"header"`
	Quick      bool         `json:"quick"`
	Creative   bool         `json:"creative"`
	Technical  bool         `json:"technical"`
	Wage       int          `json:"wage"`
	Age        int          `json:"age"`
	Avatar     PlayerAvatar `json:"avatar"`
	// ContractYearsRemaining counts down by one every season (see AdvancePlayerContract) - once
	// it hits zero the player is released to free agency (TeamId becomes unset - see
	// GetFreeAgents) rather than staying with their club forever.
	ContractYearsRemaining int `json:"contractYearsRemaining"`
	// ContractExpiring is true for another club's player in the last season of his contract, set by
	// RedactFinancials when it hides his contract years (openspec/specs/pre-contracts).
	ContractExpiring bool `json:"contractExpiring,omitempty"`
	// ReleaseClause is the public fee any club may pay to sign him without his club's consent, and
	// ReleaseClausePayableAt when it can first be paid (openspec/specs/release-clauses); both nil
	// without a clause, and only read where the player is loaded with playerColumnsSql.
	ReleaseClause          *int64     `json:"releaseClause,omitempty"`
	ReleaseClausePayableAt *time.Time `json:"releaseClausePayableAt,omitempty"`
	// BirthCountry is the country whose name pool generated this player (see doCreatePlayers) -
	// independent of TeamId's current country, since the transfer market has no country
	// restriction and a player can end up anywhere.
	BirthCountry string `json:"birthCountry,omitempty"`
	// FreeAgentSeasons and LastTeamId are only meaningful for a free agent (TeamId == 0) - see
	// SignFreeAgent, which charges LastTeamId a transfer premium when FreeAgentSeasons is still 0
	// (still within the player's first season of free agency). LastTeamId is 0 (coalesced, like
	// TeamId) when the player has never been released - e.g. still on their original team, or
	// unassigned by AssignTeam reassigning a CPU squad rather than a real release.
	FreeAgentSeasons int   `json:"freeAgentSeasons,omitempty"`
	LastTeamId       int64 `json:"lastTeamId,omitempty"`
	// ShirtNumber is 0 for a player with none assigned yet (an academy prospect not yet promoted,
	// or a free agent - see common.nextFreeShirtNumber, assigned whenever a player joins a team's
	// senior roster and cleared whenever they leave it).
	ShirtNumber int `json:"shirtNumber,omitempty"`
	// Stamina (0-99) is how well this player holds up physically within a single match - read
	// every simulated minute by engine.StaminaSimulation to fatigue a tired player's effective
	// skills, and drifted after each match by engine.updateTeamPlayersAfterMatch based on minutes
	// played and age (older players recover stamina more slowly - see engine.ageTrendForStamina).
	// Unlike the seven core skills it isn't generated inside the 15-40/35-75 min/max bands those
	// use (see doCreatePlayers) - see SeniorStaminaMin/Max and AcademyStaminaMin/Max instead.
	Stamina int `json:"stamina"`
	// Form (0-99) is a slower-moving measure of recent performances. It doesn't affect how a
	// player plays or his match rating (see ComputePlayerRatings) - the rating moves it, not the
	// other way round. Unlike Stamina it doesn't change minute-by-minute, only after a match finishes (see
	// engine.updateTeamPlayersAfterMatch), rising when the player plays (more so with a good
	// rating) and decaying when they sit out. Every new player starts at the neutral midpoint, 50
	// (see doCreatePlayers/InjectFreeAgents).
	Form int `json:"form"`
	// AverageSkill is this player's seven core skills reduced to a single 0-99 figure (see
	// common.AverageSkill) - only populated by callers that need it to survive RedactSkills (see
	// GetTeamDetailsById and PlayerDetails.AverageSkill, which redundantly recomputes its own copy
	// since it embeds Player rather than reading this field).
	AverageSkill int `json:"averageSkill,omitempty"`
}

// RedactFinancials zeroes the wage and contract terms of a player who currently belongs to a
// team - a rostered player's finances are only the owning team's business (see
// GetTeamDetailsById and handlers.PlayerDetailsHandler), unlike a free agent's wage, which stays
// visible everywhere since it's exactly the figure a prospective signer needs to see.
func (p *Player) RedactFinancials() {
	if p.TeamId == 0 {
		return
	}
	// The last season of a contract stays public (openspec/specs/pre-contracts "Last-season
	// marker"): other clubs need it to offer him a pre-contract.
	p.ContractExpiring = p.ContractYearsRemaining == 1
	p.Wage = 0
	p.ContractYearsRemaining = 0
}

// RedactSkills zeroes every attribute that reveals a player's quality (the seven core ratings
// plus the four boolean traits) unless at least one of three things is true: the player is a
// free agent (TeamId == 0 - a prospective signer needs to see exactly what they'd be getting),
// transferListed is true (the selling club put this specific player up for transfer, so a buyer
// needs the same information), or the caller already knows the viewer owns this player's team and
// simply never calls RedactSkills at all (see GetTeamDetailsById and handlers.
// PlayerDetailsHandler, both of which skip this call entirely for isOwner - RedactSkills only
// ever runs on the "someone else's rostered player" path).
func (p *Player) RedactSkills(transferListed bool) {
	if p.TeamId == 0 || transferListed {
		return
	}
	p.GoalKeeper = 0
	p.Defense = 0
	p.Lateral = 0
	p.Pass = 0
	p.Midfield = 0
	p.Dribbling = 0
	p.Shot = 0
	p.Header = false
	p.Quick = false
	p.Creative = false
	p.Technical = false
	p.Stamina = 0
	p.Form = 0
}

// AverageSkill reduces a player's seven core skill ratings to a single 0-99 headline figure,
// rounded to the nearest point - mirrors the frontend's client-side overallRating (see
// utils/playerRating.ts), kept in sync by hand since nothing shares this formula between the Go
// engine and the client. Callers that need this to survive RedactSkills (see
// handlers.PlayerDetailsHandler) must compute it before calling that.
func AverageSkill(p Player) int {
	total := p.GoalKeeper + p.Defense + p.Lateral + p.Pass + p.Midfield + p.Dribbling + p.Shot
	return int(math.Round(float64(total) / 7))
}

// TeamTransaction is one entry in a team's financial ledger.
type TeamTransaction struct {
	Id          int64     `json:"id"`
	TeamId      int64     `json:"teamId"`
	Amount      int64     `json:"amount"` // positive = income, negative = expense
	Reason      int       `json:"reason"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	// PlayerId is nil for a team-level transaction (wages, ticket revenue, prize money,
	// sponsorship, stadium upgrades, admin adjustments) - only set for one about a specific
	// player (a transfer, a release). See GetPlayerTransactions.
	PlayerId *int64 `json:"playerId,omitempty"`
}

// FreeAgentListing is a free agent as shown on the transfer market's free agents tab - the bare
// Player plus their last club's name (if any), resolved once here rather than making the
// frontend look it up per row. See GetFreeAgentListings.
type FreeAgentListing struct {
	Player
	LastTeamName string `json:"lastTeamName,omitempty"`
	// Caps is this player's international caps count (see GetPlayerCapsBatch) - resolved once
	// here, batched across the whole listing, rather than per player, so the free agents tab can
	// offer a "minimum caps" filter without looking each one up itself.
	Caps int `json:"caps"`
}

// TransferListing is a player up for transfer, bought instantly at AskingPrice (no bidding).
type TransferListing struct {
	ListingId       int64     `json:"listingId"`
	Player          Player    `json:"player"`
	SellingTeamId   int64     `json:"sellingTeamId"`
	SellingTeamName string    `json:"sellingTeamName"`
	AskingPrice     int64     `json:"askingPrice"`
	ListedAt        time.Time `json:"listedAt"`
	// Caps is this listing's player's international caps count (see GetPlayerCapsBatch) - lets
	// the transfer market's search offer a "minimum caps" filter without the frontend looking
	// each listed player up individually.
	Caps int `json:"caps"`
}

type PlayerAvatar struct {
	AccessoriesType string `json:"accessoriesType"`
	AvatarStyle     string `json:"avatarStyle"`
	ClotheColor     string `json:"clotheColor"`
	ClotheType      string `json:"clotheType"`
	EyeType         string `json:"eyeType"`
	EyebrowType     string `json:"eyebrowType"`
	FacialHairColor string `json:"facialHairColor"`
	FacialHairType  string `json:"facialHairType"`
	GraphicType     string `json:"graphicType"`
	HairColor       string `json:"hairColor"`
	HatColor        string `json:"hatColor"`
	MouthType       string `json:"mouthType"`
	SkinColor       string `json:"skinColor"`
	TopType         string `json:"topType"`
}

type TeamDetails struct {
	Team        *Team       `json:"team"`
	Players     []*Player   `json:"players"`
	LastMatches []WeekMatch `json:"lastMatches"`
	NextMatches []WeekMatch `json:"nextMatches"`
	// Stadium is this team's ground - name, capacity, and its most recently played home match (see
	// GetTeamDetailsById). Public information (unlike the rest of Stadium's fields - ticket price,
	// pending upgrades - which stay owner-only on GetFinancesHandler), the same way a real club's
	// stadium and attendance figures are.
	Stadium *StadiumSummary `json:"stadium,omitempty"`
	// Fanbase is this team's current 0-99 supporter-base index (see GetFanbase) - public, same
	// visibility tier as Stadium's own public fields, the way a real club's attendance figures
	// are public knowledge.
	Fanbase int `json:"fanbase"`
	// Unavailability lists the squad's currently suspended/injured players - only filled in for
	// the caller's own team (see GetFullTeamDetailsByUser), where the lineup editors use it to
	// flag who can't be fielded. Never cached (unlike GetTeamDetails' views), since it changes
	// after every match the team plays.
	Unavailability []PlayerUnavailability `json:"unavailability,omitempty"`
	// TransferListedPlayerIds is the set of the squad's own players currently listed for transfer
	// (see GetTransferListedPlayerIdsForTeam) - only filled in for the caller's own team, same as
	// Unavailability, so the squad and lineup pages can flag them the same way an injured/
	// suspended player is flagged.
	TransferListedPlayerIds []int64 `json:"transferListedPlayerIds,omitempty"`
	// Trainer is this team's current head coach (see GetTeamTrainer), or nil if it hasn't hired
	// one - public, same visibility tier as Stadium/Fanbase (a real club's manager is public
	// knowledge too), unlike the owner-only hire/fire actions themselves. Never populated for a
	// national team (see getNationalTeamDetails), which has no manager to hire one.
	Trainer *Trainer `json:"trainer,omitempty"`
	// LoanedOut lists the players this club has out on loan (see GetLoanedOutPlayers) and LoanedIn
	// the players in Players who are only there on loan (see GetLoanedInPlayers). Public, like the
	// loan market and player histories they come from, and always arrays (empty for a national
	// team, which has no loans).
	LoanedOut []LoanedOutPlayer `json:"loanedOut"`
	LoanedIn  []LoanedInPlayer  `json:"loanedIn"`
}

// PlayerUnavailability is one player's current suspension or injury - Reason is
// UNAVAILABLE_SUSPENSION or UNAVAILABLE_INJURY.
type PlayerUnavailability struct {
	PlayerId         int64 `json:"playerId"`
	Reason           int   `json:"reason"`
	MatchesRemaining int   `json:"matchesRemaining"`
}

type MatchSimulation struct {
	Id         int64
	DivisionId int64
	// Minute is the clock minute being simulated (1-90); during a half's added time it stays at
	// 45 or 90 while AddedMinute counts 1..N (0 in regulation). PlayedMinutes counts every minute
	// simulated so far, added minutes included.
	Minute        int
	AddedMinute   int
	PlayedMinutes int
	// FirstHalfAddedTime/SecondHalfAddedTime are each half's added minutes, fixed when its
	// regulation minutes end (0 for a half never reached).
	FirstHalfAddedTime  int
	SecondHalfAddedTime int
	Home                int64
	Away                int64
	HomeLineup          *Lineup
	AwayLineup          *Lineup
	HomeGoals           int
	AwayGoals           int
	//Players
	HomePlayers           *map[int64]Player
	AwayPlayers           *map[int64]Player
	HomePlayersByPosition *map[int][]int64
	AwayPlayersByPosition *map[int][]int64
	//EffectiveHomePlayers/EffectiveAwayPlayers: a per-minute fatigue-adjusted copy of
	//HomePlayers/AwayPlayers, rebuilt every minute by engine.StaminaSimulation (the pipeline step
	//that runs right before InitialSimulation) from each player's Stamina and their minutes played
	//so far this match (see HomeMinutesByPosition/AwayMinutesByPosition). InitialSimulation's
	//calculateQua and ScorerSimulation's direct per-player skill reads both use these instead of
	//HomePlayers/AwayPlayers, so a tired player's reduced skills consistently affect both a team's
	//aggregate zone qualifications and any individual chance they're personally involved in.
	EffectiveHomePlayers *map[int64]Player
	EffectiveAwayPlayers *map[int64]Player
	//Qualifications
	HomeQua *TeamQualification
	AwayQua *TeamQualification
	//HomeMentality/AwayMentality: each side's mentality this minute (MENTALITY_*), initialised from
	//its kickoff lineup and read every minute by InitialSimulation (ratings), ScorerSimulation
	//(chance probability) and StaminaSimulation (fatigue).
	HomeMentality int
	AwayMentality int
	//RivalryLevel: the pair's rivalry level at kickoff (RIVALRY_*, see common/rivalries.go) - 0 in
	//friendlies. Read by DisciplineSimulation (card rate) and stored with the result.
	RivalryLevel int
	//HomeOrders/AwayOrders: each side's conditional match orders (see common/lineuporders.go) and
	//which of them have been spent, applied by engine.SubstitutionSimulation after the scheduled
	//substitutions.
	HomeOrders      []LineupOrder
	AwayOrders      []LineupOrder
	HomeOrdersFired []bool
	AwayOrdersFired []bool
	//Calculations
	MidfieldDiff         float32 // home midfield vs away midfield difference
	HomeAttackRightDiff  float32
	HomeAttackLeftDiff   float32
	HomeAttackCenterDiff float32
	AwayAttackRightDiff  float32
	AwayAttackLeftDiff   float32
	AwayAttackCenterDiff float32
	//Events
	Events *[]Event
	//Instants
	MatchInstants *[]MatchInstant
	//Discipline: players with a yellow card this match (to detect a second yellow -> red),
	//and players sent off (red card or second yellow), already removed from *PlayersByPosition.
	YellowCards map[int64]bool
	SentOff     map[int64]bool
	//Unavailability: suspensions/injuries picked up this match, to persist once it's over
	//(see engine.UpdatePlayedMatch) so they carry into the player's next matches.
	PendingUnavailabilities []PendingUnavailability
	//Substitutions: each team's manager-scheduled in-match changes (see
	//GetDefaultLineupSubstitutions), applied at their chosen minute by
	//engine.SubstitutionSimulation and then left in place - already-applied entries aren't
	//removed, since each minute of the simulation loop runs exactly once.
	HomeSubstitutions []LineupSubstitution
	AwaySubstitutions []LineupSubstitution
	//SubstitutionsUsed: how many of a team's MaxScheduledSubstitutions have actually been spent
	//so far this match - incremented by engine.SubstitutionSimulation for a scheduled swap that
	//fires and by engine.InjurySimulation for an emergency one, so whichever runs later in a
	//given minute sees the other's count. Caps how many more emergency substitutions an injury
	//can trigger - once a team has none left, an injured player simply isn't replaced.
	HomeSubstitutionsUsed int
	AwaySubstitutionsUsed int
	//SubstitutedOff: each team's players taken off this match by a substitution (scheduled or
	//an injury replacement) - never allowed back on (see engine.bestEligibleBenchPlayer). Created
	//lazily by the engine (engine.sideOf), so a zero-value MatchSimulation is fine.
	HomeSubstitutedOff map[int64]bool
	AwaySubstitutedOff map[int64]bool
	//UnavailablePlayers: each team's players already ruled out for this match before kickoff
	//(suspension/injury carried over from a previous match - see engine.excludeUnavailablePlayers),
	//kept here so engine.InjurySimulation can skip them too when picking a bench player to bring
	//on for someone injured mid-match, without re-querying the database.
	HomeUnavailablePlayers map[int64]bool
	AwayUnavailablePlayers map[int64]bool
	//MinutesByPosition: per player, minutes spent at each tactical position they occupied
	//during this match (playerId -> position -> minutes) - see engine.trackMinutesByPosition.
	//Consumed right after the match by engine.updatePlayerSkillsAfterMatch, which gates and
	//scales that player's skill growth on having actually played and how many minutes.
	HomeMinutesByPosition map[int64]map[int]int
	AwayMinutesByPosition map[int64]map[int]int
	//StoredRatings: the match's rated players' ratings (see PostMatchPayload.Ratings), set only
	//when post-match effects are applied from a payload that carries them - the source of Form's
	//performance adjustment (see engine.matchRatingsForProgression).
	StoredRatings map[int64]float64
	//IsFriendly: whether this match is a friendly rather than a league or cup fixture - a red
	//card picked up in a friendly (see engine.DisciplineSimulation) carries no suspension at
	//all, for a friendly or otherwise, since a friendly has no competitive stakes to sanction.
	IsFriendly bool
	//HomeAdvantage: the multiplier applied every minute to the home side's nine zone ratings
	//(see engine.InitialSimulation and engine.homeAdvantageMultiplier), set before kickoff from
	//the crowd drawn for the match. 1 for World Cup fixtures; 0 (unset) also counts as 1.
	HomeAdvantage float32
}

// PendingUnavailability is a suspension or injury picked up during a match, not yet
// persisted to the player_unavailability table.
type PendingUnavailability struct {
	PlayerId int64
	Reason   int
	Matches  int
}

// TeamQualification is a team's per-zone strength for one minute of simulation - the same
// numbers engine.processChance reads directly (attack on one flank vs the opponent's mirrored
// defense on the other, see its own comment) to decide whether a chance becomes a goal, so this
// is genuinely what's "used for calculating the probability of goal", not a display-only
// summary. Goalkeeper is declared but never populated (goalkeeping strength is read live from
// the starting keeper's own Player.GoalKeeper instead - see processChance) - left here rather
// than removed since MatchInstant already persists this shape to the database.
type TeamQualification struct {
	Goalkeeper    float32 `json:"goalkeeper"`
	LeftDefense   float32 `json:"leftDefense"`
	RightDefense  float32 `json:"rightDefense"`
	Defense       float32 `json:"defense"`
	LeftMidfield  float32 `json:"leftMidfield"`
	RightMidfield float32 `json:"rightMidfield"`
	Midfield      float32 `json:"midfield"`
	LeftAttack    float32 `json:"leftAttack"`
	// Tag deliberately matches the field's own historical (typo'd) spelling rather than
	// correcting it to "rightAttack" - this struct is stored as JSON in match_instants, and
	// already-simulated matches' stored rows use this exact key (from Go's untagged default
	// marshaling before this struct had tags at all); a differently-spelled tag would silently
	// fail to unmarshal that old data back into this field (Go's case-insensitive JSON matching
	// bridges a casing difference, not a spelling one), leaving it looking like a real zero
	// instead of missing data.
	RightAttach float32 `json:"rightAttach"`
	Attack      float32 `json:"attack"`
}

type MatchInstant struct {
	MatchId int64 `json:"matchId"`
	Minute  int   `json:"minute"`
	// AddedMinute is 1..N inside a half's added time (Minute then reads 45 or 90), 0 otherwise.
	AddedMinute int `json:"addedMinute"`
	HomeGoals   int `json:"homeGoals"`
	AwayGoals   int `json:"awayGoals"`
	//Qualifications
	HomeQua TeamQualification `json:"homeQua"`
	AwayQua TeamQualification `json:"awayQua"`
	//Calculations
	MidfieldDiff         float32 `json:"midfieldDiff"`
	HomeAttackRightDiff  float32 `json:"homeAttackRightDiff"`
	HomeAttackLeftDiff   float32 `json:"homeAttackLeftDiff"`
	HomeAttackCenterDiff float32 `json:"homeAttackCenterDiff"`
	AwayAttackRightDiff  float32 `json:"awayAttackRightDiff"`
	AwayAttackLeftDiff   float32 `json:"awayAttackLeftDiff"`
	AwayAttackCenterDiff float32 `json:"awayAttackCenterDiff"`
	// PlayerContributions is this minute's zone contribution of every player on the pitch, per
	// side - see engine.playerContributions and ComputePlayerRatings, which builds each player's
	// zone component from them. Nil for instants saved before init89_match_player_ratings.sql.
	PlayerContributions *InstantContributions `json:"playerContributions,omitempty"`
}

// InstantContributions is one minute's PlayerContribution of every player on the pitch, keyed by
// player id, one map per side. The JSON keys are deliberately short: one of these is stored per
// minute of every match.
type InstantContributions struct {
	Home map[int64]PlayerContribution `json:"h"`
	Away map[int64]PlayerContribution `json:"a"`
}

// PlayerContribution is one player's share of the zone duels his position feeds during one
// minute: Own is his own value (the same fatigue-adjusted, position-weighted figure the engine
// adds into that zone's TeamQualification - the mean of the two when he feeds two duels) and Rival
// is the opponent's per-player value in the mirrored duel(s). Both are rounded to one decimal
// before anything uses them, so the stored figures rebuild the exact same ratings.
type PlayerContribution struct {
	Position int     `json:"p"`
	Own      float32 `json:"o"`
	Rival    float32 `json:"r"`
}

type Event struct {
	EventType             int      `json:"eventType"`
	Minute                int      `json:"minute"`
	TeamOwner             int64    `json:"teamOwner"`
	TeamOwnerPlayers      []int64  `json:"teamOwnerPlayers"`
	TeamOwnerPlayersNames []string `json:"teamOwnerPlayerNames"`
	OtherTeam             int64    `json:"otherTeam"`
	OtherTeamPlayers      []int64  `json:"otherTeamPlayers"`
	OtherTeamPlayersNames []string `json:"otherTeamPlayerNames"`
	HomeGoals             int      `json:"homeGoals"`
	AwayGoals             int      `json:"awayGoals"`
	// SetPiece is SET_PIECE_NONE (open play), SET_PIECE_PENALTY or SET_PIECE_DIRECT_FREE_KICK; only
	// meaningful on GOAL / NO_GOAL events. Every GOAL / NO_GOAL event, open play included, lists the
	// defending goalkeeper in OtherTeamPlayers / OtherTeamPlayersNames (stored for league, cup and
	// friendly events alike; empty on events stored before init108_event_other_team_players.sql).
	SetPiece int `json:"setPiece"`
	// Mentality is the side's new mentality on a MENTALITY_CHANGE event; nil on every other event.
	Mentality *int `json:"mentality,omitempty"`
	// AddedMinute is 1..N for an event in a half's added time (Minute then reads 45 or 90), 0
	// otherwise.
	AddedMinute int `json:"addedMinute"`
	// AddedTime is the number of added minutes an ADDED_TIME event announces; nil on every other
	// event.
	AddedTime *int `json:"addedTime,omitempty"`
}

type WeekMatch struct {
	MatchId      int64     `json:"matchId"`
	DivisionId   int64     `json:"divisionId"`
	Season       int       `json:"season"`
	WeekNumber   int       `json:"weekNumber"`
	MatchDate    time.Time `json:"matchDate"`
	Played       bool      `json:"played"`
	InSimulation bool      `json:"inSimulation"`
	Home         int64     `json:"home"`
	HomeName     string    `json:"homeName"`
	// HomeStadiumName is the home team's stadium at the time this match is fetched (not stored
	// per-match like Weather/Attendance/etc below), so a later stadium rename is reflected on any
	// already-played match too.
	HomeStadiumName string            `json:"homeStadiumName"`
	Away            int64             `json:"away"`
	AwayName        string            `json:"awayName"`
	HomeGoals       int               `json:"homeGoals"`
	AwayGoals       int               `json:"awayGoals"`
	Events          *[]Event          `json:"events"`
	MatchInstants   *[]MatchInstant   `json:"matchInstants"`
	HomeLineup      *Lineup           `json:"homeLineup"`
	AwayLineup      *Lineup           `json:"awayLineup"`
	HomePlayers     *map[int64]Player `json:"homePlayers"`
	AwayPlayers     *map[int64]Player `json:"awayPlayers"`
	// Weather/Attendance/RefereeName/Assistant1Name/Assistant2Name are the match report's cosmetic
	// flavor (issue: Hattrick-style match report) - empty/0 for a match not yet simulated, or one
	// simulated before this existed (see resources/sql/init40_match_flavor.sql). Generated once at
	// simulation time (common.UpdateMatchFlavor) rather than per-request, so the report is stable.
	Weather        string `json:"weather,omitempty"`
	Attendance     int    `json:"attendance,omitempty"`
	RefereeName    string `json:"refereeName,omitempty"`
	Assistant1Name string `json:"assistant1Name,omitempty"`
	Assistant2Name string `json:"assistant2Name,omitempty"`
	// PlayerRatings is every player who featured, with his rating and its breakdown - the stored
	// rows, or the fallback for a match simulated before them (see LoadPlayerRatings). Only
	// populated for a played match.
	PlayerRatings []PlayerRating `json:"playerRatings,omitempty"`
	// Ratings is PlayerRatings as a player id -> rating map, rated players only (see
	// RatingsById). Keyed by player id (encoding/json marshals an integer-keyed map as a
	// string-keyed JSON object, same as HomePlayers/AwayPlayers above).
	Ratings *map[int64]float64 `json:"ratings,omitempty"`
	// Rivalry is the match's rivalry level (RIVALRY_*): the one it was played at, or the pair's
	// current one before kickoff - see GetFixtureRivalry. Only WeekMatchById fills it.
	Rivalry int `json:"rivalry"`
}
