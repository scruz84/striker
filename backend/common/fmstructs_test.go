package common

import "testing"

func TestRedactFinancials_ZeroesRosteredPlayer(t *testing.T) {
	p := Player{TeamId: 42, Wage: 500, ContractYearsRemaining: 2}
	p.RedactFinancials()
	if p.Wage != 0 || p.ContractYearsRemaining != 0 {
		t.Fatalf("expected wage and contract to be zeroed, got wage=%d contractYearsRemaining=%d", p.Wage, p.ContractYearsRemaining)
	}
}

func TestRedactFinancials_LeavesFreeAgentUntouched(t *testing.T) {
	p := Player{TeamId: 0, Wage: 500, ContractYearsRemaining: 0}
	p.RedactFinancials()
	if p.Wage != 500 {
		t.Fatalf("expected a free agent's wage to stay visible, got %d", p.Wage)
	}
}

func fullSkillPlayer(teamId int64) Player {
	return Player{
		TeamId: teamId, GoalKeeper: 80, Defense: 70, Lateral: 60, Pass: 90, Midfield: 55,
		Dribbling: 65, Shot: 75, Header: true, Quick: true, Creative: true, Technical: true,
	}
}

func TestRedactSkills_ZeroesRosteredUnlistedPlayer(t *testing.T) {
	p := fullSkillPlayer(42)
	p.RedactSkills(false)
	if p.GoalKeeper != 0 || p.Defense != 0 || p.Lateral != 0 || p.Pass != 0 || p.Midfield != 0 ||
		p.Dribbling != 0 || p.Shot != 0 || p.Header || p.Quick || p.Creative || p.Technical {
		t.Fatalf("expected every skill/trait to be zeroed, got %+v", p)
	}
}

func TestRedactSkills_LeavesTransferListedPlayerUntouched(t *testing.T) {
	p := fullSkillPlayer(42)
	want := p
	p.RedactSkills(true)
	if p != want {
		t.Fatalf("expected a transfer-listed player's skills to stay visible, got %+v, want %+v", p, want)
	}
}

func TestRedactSkills_LeavesFreeAgentUntouched(t *testing.T) {
	p := fullSkillPlayer(0)
	want := p
	p.RedactSkills(false)
	if p != want {
		t.Fatalf("expected a free agent's skills to stay visible, got %+v, want %+v", p, want)
	}
}
