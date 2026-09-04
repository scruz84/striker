package common

import (
	"database/sql"
	"errors"
	"math/rand"
	"sort"
	"time"
)

var ErrGroupNotFound = errors.New("group not found")
var ErrInvalidInviteCode = errors.New("invalid invite code")
var ErrAlreadyInGroup = errors.New("already a member of this group")
var ErrNotGroupOwner = errors.New("only the group owner can do this")

// Group is a friend group - a lightweight social leaderboard among teams its members already
// own in the shared world, NOT an isolated private league with its own teams/fixtures (that
// would need generating a whole separate league on demand - a much bigger change). See
// GetGroupLeaderboard.
type Group struct {
	GroupId    int64     `json:"groupId"`
	Name       string    `json:"name"`
	OwnerId    int64     `json:"ownerId"`
	InviteCode string    `json:"inviteCode"`
	CreatedAt  time.Time `json:"createdAt"`
}

// GroupMember is one leaderboard row - a group member's user and (if they have one) team.
type GroupMember struct {
	UserId     int64  `json:"userId"`
	UserName   string `json:"userName"`
	TeamId     *int64 `json:"teamId"`
	TeamName   string `json:"teamName"`
	Country    string `json:"country"`
	LeagueName string `json:"leagueName"`
	Budget     *int64 `json:"budget"`
}

const inviteCodeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no ambiguous 0/O/1/I
const inviteCodeLength = 8

func generateInviteCode(r *rand.Rand) string {
	code := make([]byte, inviteCodeLength)
	for i := range code {
		code[i] = inviteCodeChars[r.Intn(len(inviteCodeChars))]
	}
	return string(code)
}

const InsertGroupStmtSql = "INSERT INTO groups (name, owner_id, invite_code) VALUES ($1, $2, $3) RETURNING group_id"
const InsertGroupMemberStmtSql = "INSERT INTO group_members (group_id, user_id) VALUES ($1, $2)"
const GetGroupByInviteCodeStmtSql = "SELECT group_id, name, owner_id, invite_code, created_at FROM groups WHERE invite_code = $1"
const GetGroupByIdStmtSql = "SELECT group_id, name, owner_id, invite_code, created_at FROM groups WHERE group_id = $1"
const GetGroupsForUserStmtSql = "SELECT g.group_id, g.name, g.owner_id, g.invite_code, g.created_at " +
	"FROM groups g JOIN group_members gm ON gm.group_id = g.group_id WHERE gm.user_id = $1 ORDER BY g.created_at DESC"
const GetGroupMemberUserIdsStmtSql = "SELECT user_id FROM group_members WHERE group_id = $1"
const DeleteGroupMemberStmtSql = "DELETE FROM group_members WHERE group_id = $1 AND user_id = $2"
const DeleteGroupStmtSql = "DELETE FROM groups WHERE group_id = $1"
const IsGroupMemberStmtSql = "SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2"

// CreateGroup makes a new friend group and immediately adds its creator as the first member.
func CreateGroup(dbCtx *DatabaseContext, name string, ownerId int64) (*Group, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var groupId int64
	var inviteCode string
	// A handful of retries covers the astronomically unlikely case of a collision on an
	// 8-character code drawn from a 33-character alphabet (33^8 possibilities).
	for attempt := 0; attempt < 5; attempt++ {
		inviteCode = generateInviteCode(r)
		err = tx.QueryRow(InsertGroupStmtSql, name, ownerId, inviteCode).Scan(&groupId)
		if err == nil || !isUniqueViolation(err) {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(InsertGroupMemberStmtSql, groupId, ownerId); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Group{GroupId: groupId, Name: name, OwnerId: ownerId, InviteCode: inviteCode}, nil
}

func scanGroup(row rowScanner) (*Group, error) {
	var g Group
	if err := row.Scan(&g.GroupId, &g.Name, &g.OwnerId, &g.InviteCode, &g.CreatedAt); err != nil {
		return nil, err
	}
	return &g, nil
}

// JoinGroup adds userId to the group identified by inviteCode.
func JoinGroup(dbCtx *DatabaseContext, inviteCode string, userId int64) (*Group, error) {
	group, err := scanGroup(dbCtx.Db.QueryRow(GetGroupByInviteCodeStmtSql, inviteCode))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidInviteCode
	}
	if err != nil {
		return nil, err
	}

	_, err = dbCtx.Db.Exec(InsertGroupMemberStmtSql, group.GroupId, userId)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyInGroup
		}
		return nil, err
	}
	return group, nil
}

// LeaveGroup removes userId from a group - anyone, owner included, can leave; the group itself
// (and its invite code) persists regardless of who's still in it.
func LeaveGroup(dbCtx *DatabaseContext, groupId int64, userId int64) error {
	_, err := dbCtx.Db.Exec(DeleteGroupMemberStmtSql, groupId, userId)
	return err
}

// DeleteGroup removes a group entirely (cascading to its memberships) - owner only.
func DeleteGroup(dbCtx *DatabaseContext, groupId int64, requestingUserId int64) error {
	group, err := GetGroupById(dbCtx, groupId)
	if err != nil {
		return err
	}
	if group.OwnerId != requestingUserId {
		return ErrNotGroupOwner
	}
	_, err = dbCtx.Db.Exec(DeleteGroupStmtSql, groupId)
	return err
}

func GetGroupById(dbCtx *DatabaseContext, groupId int64) (*Group, error) {
	group, err := scanGroup(dbCtx.Db.QueryRow(GetGroupByIdStmtSql, groupId))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrGroupNotFound
	}
	return group, err
}

// GetGroupsForUser lists every group userId belongs to.
func GetGroupsForUser(dbCtx *DatabaseContext, userId int64) ([]Group, error) {
	rows, err := dbCtx.Db.Query(GetGroupsForUserStmtSql, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]Group, 0)
	for rows.Next() {
		g, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, *g)
	}
	return groups, nil
}

// IsGroupMember reports whether userId belongs to groupId.
func IsGroupMember(dbCtx *DatabaseContext, groupId int64, userId int64) (bool, error) {
	var exists int
	err := dbCtx.Db.QueryRow(IsGroupMemberStmtSql, groupId, userId).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// GetGroupLeaderboard lists every member's team - name, country, current league, and budget -
// sorted richest first. Budget is the one number that's always comparable between any two teams
// regardless of which league or country they're in, unlike classification position (which only
// means something relative to a team's own league).
func GetGroupLeaderboard(dbCtx *DatabaseContext, groupId int64) ([]GroupMember, error) {
	rows, err := dbCtx.Db.Query(GetGroupMemberUserIdsStmtSql, groupId)
	if err != nil {
		return nil, err
	}
	userIds := make([]int64, 0)
	for rows.Next() {
		var userId int64
		if err := rows.Scan(&userId); err != nil {
			rows.Close()
			return nil, err
		}
		userIds = append(userIds, userId)
	}
	rows.Close()

	members := make([]GroupMember, 0, len(userIds))
	for _, userId := range userIds {
		user, err := GetUserDetails(userId, dbCtx)
		if err != nil {
			continue
		}
		member := GroupMember{UserId: userId, UserName: user.Name}
		if team, err := GetTeamByOwner(userId, dbCtx); err == nil {
			member.TeamId = &team.Id
			member.TeamName = team.TeamName
			member.Country = team.Country
			member.LeagueName = team.DivisionName
			if budget, err := GetTeamBudget(dbCtx, team.Id); err == nil {
				member.Budget = &budget
			}
		}
		members = append(members, member)
	}

	sort.SliceStable(members, func(i, j int) bool {
		bi, bj := int64(-1<<62), int64(-1<<62)
		if members[i].Budget != nil {
			bi = *members[i].Budget
		}
		if members[j].Budget != nil {
			bj = *members[j].Budget
		}
		return bi > bj
	})

	return members, nil
}
