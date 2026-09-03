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
}

type HttpResponse struct {
	Message string `json:"message"`
}

type Lineup struct {
	MatchId   int64            `json:"matchId"`
	TeamId    int64            `json:"teamId"`
	Positions []LineupPosition `json:"positions"`
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
}

type Player struct {
	Id         int64        `json:"id"`
	TeamId     int64        `json:"teamId"`
	FirstName  string       `json:"firstName"`
	LastName   string       `json:"lastName"`
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
	Avatar     PlayerAvatar `json:"avatar"`
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
}

type TeamQualification struct {
	Goalkeeper    float32
	LeftDefense   float32
	RightDefense  float32
	Defense       float32
	LeftMidfield  float32
	RightMidfield float32
	Midfield      float32
	LeftAttack    float32
	RightAttach   float32
	Attack        float32
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
}
