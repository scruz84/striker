package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

// cpu-team-management "CPU sponsor deal": a struggling Manager switches to Steady through
// SetSponsorDeal, which locks it - a later run of wins does not move it again this season.
func TestAiSponsorDeal_ManagerSwitchesOnceAndIsLocked(t *testing.T) {
	const country = "zt"
	w := newLoanWorld(t, country)
	team := w.team(country, "Strugglers", common.AiLevelManager, nil, 1_000_000)
	w.exec(`INSERT INTO classification (league_id, team_id, season, matches, points, wins, draws, losses) VALUES ($1, $2, 1, 8, 5, 1, 2, 5)`, w.league, team)

	chooseSponsorDealIfManager(w.dbCtx, w.teamOf(team))
	if deal := w.count(`SELECT sponsor_deal FROM teams WHERE team_id = $1`, team); deal != int(common.SponsorDealSteady) {
		t.Fatalf("deal %d, want Steady (%d)", deal, common.SponsorDealSteady)
	}

	w.exec(`UPDATE classification SET matches = 20, wins = 17, draws = 1 WHERE team_id = $1`, team)
	chooseSponsorDealIfManager(w.dbCtx, w.teamOf(team))
	if deal := w.count(`SELECT sponsor_deal FROM teams WHERE team_id = $1`, team); deal != int(common.SponsorDealSteady) {
		t.Errorf("deal %d after the lock, want it kept at Steady", deal)
	}

	caretaker := w.team(country, "Idle", common.AiLevelCaretaker, nil, 1_000_000)
	w.exec(`INSERT INTO classification (league_id, team_id, season, matches, points, wins, draws, losses) VALUES ($1, $2, 1, 8, 5, 1, 2, 5)`, w.league, caretaker)
	chooseSponsorDealIfManager(w.dbCtx, w.teamOf(caretaker))
	if deal := w.count(`SELECT sponsor_deal FROM teams WHERE team_id = $1`, caretaker); deal != int(common.SponsorDealBalanced) {
		t.Errorf("Caretaker deal %d, want the default Balanced", deal)
	}
}

// cpu-team-management "Manager trainer": a Manager with no trainer hires the best non-elite one
// through HireTrainer, and keeps him when a better one appears.
func TestAiTrainer_ManagerHiresTheBestNonEliteCoach(t *testing.T) {
	const country = "zu"
	w := newLoanWorld(t, country)
	w.exec(`DELETE FROM trainers WHERE team_id IS NULL`)
	elite := w.scan(`INSERT INTO trainers (first_name, last_name, skill, wage) VALUES ('Elite', 'Coach', 88, 100) RETURNING trainer_id`)
	good := w.scan(`INSERT INTO trainers (first_name, last_name, skill, wage) VALUES ('Good', 'Coach', 79, 90) RETURNING trainer_id`)
	w.exec(`INSERT INTO trainers (first_name, last_name, skill, wage) VALUES ('Fair', 'Coach', 45, 50)`)
	team := w.team(country, "Coached", common.AiLevelManager, nil, 1_000_000)

	hireTrainerIfManager(w.dbCtx, w.teamOf(team))
	if hired := w.scan(`SELECT COALESCE((SELECT trainer_id FROM trainers WHERE team_id = $1), 0)`, team); hired != good {
		t.Fatalf("hired trainer %d, want the 79 (%d)", hired, good)
	}
	if owner := w.scan(`SELECT COALESCE(team_id, 0) FROM trainers WHERE trainer_id = $1`, elite); owner != 0 {
		t.Errorf("the elite coach was taken by team %d", owner)
	}

	w.exec(`INSERT INTO trainers (first_name, last_name, skill, wage) VALUES ('Better', 'Coach', 78, 90)`)
	w.exec(`UPDATE trainers SET skill = 40 WHERE trainer_id = $1`, good)
	hireTrainerIfManager(w.dbCtx, w.teamOf(team))
	if hired := w.scan(`SELECT trainer_id FROM trainers WHERE team_id = $1`, team); hired != good {
		t.Errorf("trainer %d after a better one appeared, want the incumbent %d kept", hired, good)
	}

	caretaker := w.team(country, "Uncoached", common.AiLevelCaretaker, nil, 1_000_000)
	hireTrainerIfManager(w.dbCtx, w.teamOf(caretaker))
	if n := w.count(`SELECT COUNT(*) FROM trainers WHERE team_id = $1`, caretaker); n != 0 {
		t.Errorf("Caretaker hired %d trainers, want none", n)
	}
}
