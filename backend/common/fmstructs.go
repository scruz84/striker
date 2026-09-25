package common

import (
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
}

type LineupPosition struct {
	PlayerId int64 `json:"playerId"`
	Position int   `json:"position"`
}

type Team struct {
	Id                     int64  `json:"id"`
	OwnerId                int64  `json:"ownerId"`
	OwnerName              string `json:"ownerName"`
	Country                string `json:"country"`
	TeamName               string `json:"name"`
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
}

type Player struct {
	Id int64 `json:"id"`
	// TeamId is 0 for a free agent (team_id IS NULL in the database - see GetFreeAgents):
	// GetPlayerById coalesces the NULL to 0 rather than requiring every caller to know whether
	// the player it looked up happens to be currently unowned.
	TeamId     int64        `json:"teamId"`
	FirstName  string       `json:"firstName"`
	LastName   string       `json:"lastName"`
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
}

// RedactFinancials zeroes the wage and contract terms of a player who currently belongs to a
// team - a rostered player's finances are only the owning team's business (see
// GetTeamDetailsById and handlers.PlayerDetailsHandler), unlike a free agent's wage, which stays
// visible everywhere since it's exactly the figure a prospective signer needs to see.
func (p *Player) RedactFinancials() {
	if p.TeamId == 0 {
		return
	}
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
}

// TransferListing is a player up for transfer, bought instantly at AskingPrice (no bidding).
type TransferListing struct {
	ListingId       int64     `json:"listingId"`
	Player          Player    `json:"player"`
	SellingTeamId   int64     `json:"sellingTeamId"`
	SellingTeamName string    `json:"sellingTeamName"`
	AskingPrice     int64     `json:"askingPrice"`
	ListedAt        time.Time `json:"listedAt"`
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

type RegisterUser struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Email    string `json:"email"`
	Country  string `json:"country"`
}

type TeamDetails struct {
	Team        *Team       `json:"team"`
	Players     []*Player   `json:"players"`
	LastMatches []WeekMatch `json:"lastMatches"`
	NextMatches []WeekMatch `json:"nextMatches"`
	// Unavailability lists the squad's currently suspended/injured players - only filled in for
	// the caller's own team (see GetFullTeamDetailsByUser), where the lineup editors use it to
	// flag who can't be fielded. Never cached (unlike GetTeamDetails' views), since it changes
	// after every match the team plays.
	Unavailability []PlayerUnavailability `json:"unavailability,omitempty"`
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
	Minute     int
	Home       int64
	Away       int64
	HomeLineup *Lineup
	AwayLineup *Lineup
	HomeGoals  int
	AwayGoals  int
	//Players
	HomePlayers           *map[int64]Player
	AwayPlayers           *map[int64]Player
	HomePlayersByPosition *map[int][]int64
	AwayPlayersByPosition *map[int][]int64
	//Qualifications
	HomeQua *TeamQualification
	AwayQua *TeamQualification
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
	MatchId   int64 `json:"matchId"`
	Minute    int   `json:"minute"`
	HomeGoals int   `json:"homeGoals"`
	AwayGoals int   `json:"awayGoals"`
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
}

type WeekMatch struct {
	MatchId       int64             `json:"matchId"`
	DivisionId    int64             `json:"divisionId"`
	Season        int               `json:"season"`
	WeekNumber    int               `json:"weekNumber"`
	MatchDate     time.Time         `json:"matchDate"`
	Played        bool              `json:"played"`
	InSimulation  bool              `json:"inSimulation"`
	Home          int64             `json:"home"`
	HomeName      string            `json:"homeName"`
	Away          int64             `json:"away"`
	AwayName      string            `json:"awayName"`
	HomeGoals     int               `json:"homeGoals"`
	AwayGoals     int               `json:"awayGoals"`
	Events        *[]Event          `json:"events"`
	MatchInstants *[]MatchInstant   `json:"matchInstants"`
	HomeLineup    *Lineup           `json:"homeLineup"`
	AwayLineup    *Lineup           `json:"awayLineup"`
	HomePlayers   *map[int64]Player `json:"homePlayers"`
	AwayPlayers   *map[int64]Player `json:"awayPlayers"`
	// Weather/Attendance/RefereeName/Assistant1Name/Assistant2Name are the match report's cosmetic
	// flavor (issue: Hattrick-style match report) - empty/0 for a match not yet simulated, or one
	// simulated before this existed (see resources/sql/init40_match_flavor.sql). Generated once at
	// simulation time (common.UpdateMatchFlavor) rather than per-request, so the report is stable.
	Weather        string `json:"weather,omitempty"`
	Attendance     int    `json:"attendance,omitempty"`
	RefereeName    string `json:"refereeName,omitempty"`
	Assistant1Name string `json:"assistant1Name,omitempty"`
	Assistant2Name string `json:"assistant2Name,omitempty"`
	// Ratings is computed on read (see ComputeMatchRatings), not stored - a deterministic
	// per-player match rating (1-10) derived from the match's own events and result, only
	// populated for a played match. Keyed by player id (encoding/json marshals an integer-keyed
	// map as a string-keyed JSON object, same as HomePlayers/AwayPlayers above).
	Ratings *map[int64]float64 `json:"ratings,omitempty"`
}
