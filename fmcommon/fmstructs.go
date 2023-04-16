package fmcommon

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
	Id         int64     `json:"id"`
	TeamId     int64     `json:"teamId"`
	FirstName  string    `json:"firstName"`
	LastName   string    `json:"lastName"`
	BirthDay   time.Time `json:"birthDay"`
	GoalKeeper int       `json:"goalKeeper"`
	Defense    int       `json:"defense"`
	Lateral    int       `json:"lateral"`
	Pass       int       `json:"pass"`
	Midfield   int       `json:"midfield"`
	Dribbling  int       `json:"dribbling"`
	Shot       int       `json:"shot"`
	Header     bool      `json:"header"`
	Quick      bool      `json:"quick"`
	Creative   bool      `json:"creative"`
	Technical  bool      `json:"technical"`
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

type WeekMatch struct {
	MatchId    int64     `json:"matchId"`
	DivisionId int64     `json:"divisionId"`
	WeekNumber int       `json:"weekNumber"`
	MatchDate  time.Time `json:"matchDate"`
	Played     bool      `json:"played"`
	Home       int64     `json:"home"`
	HomeName   string    `json:"homeName"`
	Away       int64     `json:"away"`
	AwayName   string    `json:"awayName"`
	HomeGoals  int       `json:"homeGoals"`
	AwayGoals  int       `json:"awayGoals"`
}
