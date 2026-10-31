package common

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

const (
	LockUserStmtSql = "SELECT user_id FROM users WHERE user_id = $1 FOR UPDATE"
	// LockUserLastSeenStmtSql locks the row and returns last_seen_at, so an automatic erasure can
	// re-check idleness under the lock.
	LockUserLastSeenStmtSql = "SELECT last_seen_at FROM users WHERE user_id = $1 FOR UPDATE"

	GetOwnedGroupIdsStmtSql = "SELECT group_id FROM groups WHERE owner_id = $1 ORDER BY group_id"
	// The member who joined first (excluding the leaving owner) inherits the group.
	GetNextGroupOwnerStmtSql = "SELECT user_id FROM group_members WHERE group_id = $1 AND user_id <> $2 " +
		"ORDER BY joined_at, user_id LIMIT 1"
	SetGroupOwnerStmtSql   = "UPDATE groups SET owner_id = $2 WHERE group_id = $1"
	DeleteGroupByIdStmtSql = "DELETE FROM groups WHERE group_id = $1"
	DeleteUserStmtSql      = "DELETE FROM users WHERE user_id = $1"
)

// DeleteAccount erases a user in one transaction: their club is released (and reset later by the
// engine), groups they own pass to the member who joined first (or are deleted when nobody else
// is in them), and the users row is deleted - which cascades their notifications, notification
// preferences, push subscriptions and group memberships. Returns the id of the club released (0
// if they had none) so the caller can drop its caches. reason is one of the ReleaseReason*
// constants.
func DeleteAccount(dbCtx *DatabaseContext, userId int64, reason string) (int64, error) {
	return deleteAccount(dbCtx, userId, reason, nil)
}

// ErrUserActive is returned by DeleteAccountIfIdle when the user was active after the cutoff.
var ErrUserActive = errors.New("user was active since the cutoff")

// DeleteAccountIfIdle is DeleteAccount for the automatic erasure of long-idle accounts: under the
// row lock it re-reads last_seen_at and leaves the account alone (ErrUserActive) if the user came
// back after idleCutoff since the candidates were listed.
func DeleteAccountIfIdle(dbCtx *DatabaseContext, userId int64, reason string, idleCutoff time.Time) (int64, error) {
	return deleteAccount(dbCtx, userId, reason, &idleCutoff)
}

func deleteAccount(dbCtx *DatabaseContext, userId int64, reason string, idleCutoff *time.Time) (int64, error) {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var lastSeen time.Time
	if err := tx.QueryRow(LockUserLastSeenStmtSql, userId).Scan(&lastSeen); errors.Is(err, sql.ErrNoRows) {
		return 0, ErrUserNotFound
	} else if err != nil {
		return 0, err
	}
	if idleCutoff != nil && lastSeen.After(*idleCutoff) {
		return 0, ErrUserActive
	}

	teamId, err := ReleaseUserTeam(tx, userId, reason, false)
	if err != nil {
		return 0, err
	}

	rows, err := tx.Query(GetOwnedGroupIdsStmtSql, userId)
	if err != nil {
		return 0, err
	}
	groupIds := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		groupIds = append(groupIds, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, groupId := range groupIds {
		var next int64
		err := tx.QueryRow(GetNextGroupOwnerStmtSql, groupId, userId).Scan(&next)
		if errors.Is(err, sql.ErrNoRows) {
			if _, err := tx.Exec(DeleteGroupByIdStmtSql, groupId); err != nil {
				return 0, err
			}
			continue
		}
		if err != nil {
			return 0, err
		}
		if _, err := tx.Exec(SetGroupOwnerStmtSql, groupId, next); err != nil {
			return 0, err
		}
	}

	if _, err := tx.Exec(DeleteUserStmtSql, userId); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return teamId, nil
}

// DeleteConfirmationMatches reports whether typed is the confirmation DeleteAccount's endpoint
// requires: the club's exact name (case-sensitive) for a user with a club, or DELETE / BORRAR
// (any case) for one without.
func DeleteConfirmationMatches(typed string, teamName string) bool {
	typed = strings.TrimSpace(typed)
	if teamName != "" {
		return typed == teamName
	}
	return strings.EqualFold(typed, "DELETE") || strings.EqualFold(typed, "BORRAR")
}

// --- account page --------------------------------------------------------------------------

// AccountInfo is what the Account page shows: the profile plus the inactivity rule's inputs.
type AccountInfo struct {
	Name            string     `json:"name"`
	Email           string     `json:"email"`
	Provider        string     `json:"provider"`
	Country         string     `json:"country"`
	CreatedAt       time.Time  `json:"createdAt"`
	LastSeenAt      time.Time  `json:"lastSeenAt"`
	TermsVersion    *int       `json:"termsVersion"`
	TermsAcceptedAt *time.Time `json:"termsAcceptedAt"`
	TeamName        string     `json:"teamName"`
	// The thresholds, so the page states the same numbers the engine applies.
	ReleaseAfterDays int `json:"releaseAfterDays"`
	ErasureAfterDays int `json:"erasureAfterDays"`
}

const GetAccountInfoStmtSql = "SELECT u.name, u.email, u.provider, COALESCE(u.country, ''), u.created_at, u.last_seen_at, " +
	"u.terms_version, u.terms_accepted_at, COALESCE(t.name, '') FROM users u LEFT JOIN teams t ON t.owner_id = u.user_id WHERE u.user_id = $1"

// GetAccountInfo reads the Account page's data; ErrUserNotFound if the user is gone.
func GetAccountInfo(dbCtx *DatabaseContext, userId int64) (*AccountInfo, error) {
	var info AccountInfo
	var termsVersion sql.NullInt64
	var termsAt sql.NullTime
	err := dbCtx.Db.QueryRow(GetAccountInfoStmtSql, userId).Scan(&info.Name, &info.Email, &info.Provider, &info.Country,
		&info.CreatedAt, &info.LastSeenAt, &termsVersion, &termsAt, &info.TeamName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if termsVersion.Valid {
		v := int(termsVersion.Int64)
		info.TermsVersion = &v
	}
	if termsAt.Valid {
		info.TermsAcceptedAt = &termsAt.Time
	}
	info.ReleaseAfterDays = InactivityReleaseDays
	info.ErasureAfterDays = AccountErasureDays
	return &info, nil
}

// --- data export ---------------------------------------------------------------------------

type ExportedProfile struct {
	Name            string     `json:"name"`
	Email           string     `json:"email"`
	Provider        string     `json:"provider"`
	Country         string     `json:"country"`
	Language        string     `json:"language"`
	IsAdmin         bool       `json:"isAdmin"`
	CreatedAt       time.Time  `json:"createdAt"`
	LastSeenAt      time.Time  `json:"lastSeenAt"`
	TermsVersion    *int       `json:"termsVersion"`
	TermsAcceptedAt *time.Time `json:"termsAcceptedAt"`
}

type ExportedClub struct {
	TeamId      int64  `json:"teamId"`
	Name        string `json:"name"`
	Country     string `json:"country"`
	StadiumName string `json:"stadiumName"`
}

type ExportedPushSubscription struct {
	Endpoint  string    `json:"endpoint"`
	CreatedAt time.Time `json:"createdAt"`
}

type ExportedGroup struct {
	Name     string    `json:"name"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joinedAt"`
}

type ExportedNotification struct {
	Type      int       `json:"type"`
	Message   string    `json:"message"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"createdAt"`
}

// UserDataExport is everything stored about one user (right of access / portability). Push
// subscription keys (p256dh, auth) are left out: they are secrets, not information about the person.
type UserDataExport struct {
	ExportedAt              time.Time                  `json:"exportedAt"`
	Profile                 ExportedProfile            `json:"profile"`
	Club                    *ExportedClub              `json:"club"`
	NotificationPreferences NotificationPreferences    `json:"notificationPreferences"`
	PushSubscriptions       []ExportedPushSubscription `json:"pushSubscriptions"`
	Groups                  []ExportedGroup            `json:"groups"`
	Notifications           []ExportedNotification     `json:"notifications"`
}

const (
	exportProfileStmtSql = "SELECT name, email, provider, COALESCE(country, ''), COALESCE(language, ''), is_admin, created_at, last_seen_at, " +
		"terms_version, terms_accepted_at FROM users WHERE user_id = $1"
	exportClubStmtSql   = "SELECT team_id, name, country, COALESCE(stadium_name, '') FROM teams WHERE owner_id = $1"
	exportPushStmtSql   = "SELECT endpoint, created_at FROM push_subscriptions WHERE user_id = $1 ORDER BY created_at"
	exportGroupsStmtSql = "SELECT g.name, CASE WHEN g.owner_id = $1 THEN 'owner' ELSE 'member' END, m.joined_at " +
		"FROM group_members m JOIN groups g ON g.group_id = m.group_id WHERE m.user_id = $1 ORDER BY m.joined_at"
	exportNotificationsStmtSql = "SELECT notification_type, message, read, created_at FROM notifications WHERE user_id = $1 ORDER BY created_at"
)

// ExportUserData collects everything stored about userId; ErrUserNotFound if the user is gone.
func ExportUserData(dbCtx *DatabaseContext, userId int64) (*UserDataExport, error) {
	out := &UserDataExport{
		ExportedAt:        time.Now().UTC(),
		PushSubscriptions: []ExportedPushSubscription{},
		Groups:            []ExportedGroup{},
		Notifications:     []ExportedNotification{},
	}
	var termsVersion sql.NullInt64
	var termsAt sql.NullTime
	err := dbCtx.Db.QueryRow(exportProfileStmtSql, userId).Scan(&out.Profile.Name, &out.Profile.Email, &out.Profile.Provider,
		&out.Profile.Country, &out.Profile.Language, &out.Profile.IsAdmin, &out.Profile.CreatedAt, &out.Profile.LastSeenAt,
		&termsVersion, &termsAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if termsVersion.Valid {
		v := int(termsVersion.Int64)
		out.Profile.TermsVersion = &v
	}
	if termsAt.Valid {
		out.Profile.TermsAcceptedAt = &termsAt.Time
	}

	var club ExportedClub
	switch err := dbCtx.Db.QueryRow(exportClubStmtSql, userId).Scan(&club.TeamId, &club.Name, &club.Country, &club.StadiumName); {
	case err == nil:
		out.Club = &club
	case !errors.Is(err, sql.ErrNoRows):
		return nil, err
	}

	prefs, err := GetNotificationPreferences(dbCtx, userId)
	if err != nil {
		return nil, err
	}
	out.NotificationPreferences = *prefs

	rows, err := dbCtx.Db.Query(exportPushStmtSql, userId)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var s ExportedPushSubscription
		if err := rows.Scan(&s.Endpoint, &s.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		out.PushSubscriptions = append(out.PushSubscriptions, s)
	}
	rows.Close()

	rows, err = dbCtx.Db.Query(exportGroupsStmtSql, userId)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var g ExportedGroup
		if err := rows.Scan(&g.Name, &g.Role, &g.JoinedAt); err != nil {
			rows.Close()
			return nil, err
		}
		out.Groups = append(out.Groups, g)
	}
	rows.Close()

	rows, err = dbCtx.Db.Query(exportNotificationsStmtSql, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var n ExportedNotification
		if err := rows.Scan(&n.Type, &n.Message, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		out.Notifications = append(out.Notifications, n)
	}
	return out, rows.Err()
}

// --- terms ---------------------------------------------------------------------------------

const AcceptTermsStmtSql = "UPDATE users SET terms_version = $2, terms_accepted_at = now() WHERE user_id = $1"
const GetTermsVersionStmtSql = "SELECT terms_version FROM users WHERE user_id = $1"
const ClearReleasedTeamNameStmtSql = "UPDATE users SET released_team_name = NULL WHERE user_id = $1"

// AcceptTerms records that userId accepted the Privacy Policy and Terms at version, confirming
// they are at least MinimumAge. version must be CurrentTermsVersion.
func AcceptTerms(dbCtx *DatabaseContext, userId int64, version int, ageConfirmed bool) error {
	if version != CurrentTermsVersion {
		return ErrInvalidTermsVersion
	}
	if !ageConfirmed {
		return ErrAgeNotConfirmed
	}
	res, err := dbCtx.Db.Exec(AcceptTermsStmtSql, userId, version)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrUserNotFound
	}
	return nil
}

// HasAcceptedCurrentTerms reports whether userId accepted at least CurrentTermsVersion.
func HasAcceptedCurrentTerms(dbCtx *DatabaseContext, userId int64) (bool, error) {
	var version sql.NullInt64
	err := dbCtx.Db.QueryRow(GetTermsVersionStmtSql, userId).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrUserNotFound
	}
	if err != nil {
		return false, err
	}
	return version.Valid && int(version.Int64) >= CurrentTermsVersion, nil
}
