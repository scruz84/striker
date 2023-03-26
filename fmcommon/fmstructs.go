package fmcommon

import (
	"time"
)

type Classification struct {
	Positions []ClassificationPosition `json:"positions"`
}

type ClassificationPosition struct {
	Position int    `json:"position,omitempty"`
	TeamId   int64  `json:"teamId,omitempty"`
	TeamName string `json:"teamName,omitempty"`
	Points   int    `json:"points,omitempty"`
}

type Division struct {
	Id             int64          `json:"id"`
	Name           string         `json:"name"`
	Classification Classification `json:"classification"`
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
	Id           int64  `json:"id"`
	OwnerId      int64  `json:"ownerId"`
	Country      string `json:"country"`
	TeamName     string `json:"name"`
	DivisionId   int64  `json:"divisionId"`
	DivisionName string `json:"divisionName"`
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
	Team    *Team     `json:"team"`
	Players []*Player `json:"players"`
}

type WeekMatch struct {
	MatchId    int64     `json:"matchId"`
	DivisionId int64     `json:"divisionId"`
	WeekNumber int       `json:"weekNumber"`
	MatchDate  time.Time `json:"matchDate"`
	Home       int64     `json:"home"`
	Away       int64     `json:"away"`
	HomeGoals  int       `json:"HomeGoals"`
	AwayGoals  int       `json:"awayGoals"`
}
