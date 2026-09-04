package common

import log "github.com/sirupsen/logrus"

// Notifier is the pluggable interface for delivering a notification outside the in-app feed -
// email, push, or anything else. The only implementation here is LogNotifier: this environment
// has no email/push service configured (no SMTP/SES credentials, no push provider), so "sending"
// means logging what would have been sent rather than claiming delivery this code can't verify.
// Swapping in a real channel is a matter of implementing this interface (e.g. an SMTP-backed
// EmailNotifier) and pointing DefaultNotifier at it - every call site goes through that one
// variable, so nothing else has to change.
type Notifier interface {
	Send(userId int64, channel string, message string) error
}

// LogNotifier logs what it would send instead of actually sending anything.
type LogNotifier struct{}

func (LogNotifier) Send(userId int64, channel string, message string) error {
	log.Infof("[LogNotifier] would send %s notification to user %d: %s", channel, userId, message)
	return nil
}

// DefaultNotifier is the Notifier NotifyTeamOwner delivers through when a user has enabled a
// channel (see notification_preferences / GetNotificationPreferences).
var DefaultNotifier Notifier = LogNotifier{}
