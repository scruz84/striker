package common

import "testing"

func TestRenderNotificationMessageSpanish(t *testing.T) {
	msg := NotificationMessage{
		Text:   "Sold Messi to Barca",
		Key:    "notifications.messages.transferSold",
		Params: map[string]any{"player": "Messi", "team": "Barca"},
	}

	got := renderNotificationMessage(msg, "es")
	want := "Vendido Messi a Barca"
	if got != want {
		t.Errorf("renderNotificationMessage(es) = %q, want %q", got, want)
	}
}

func TestRenderNotificationMessageFallsBackToEnglish(t *testing.T) {
	msg := NotificationMessage{
		Text:   "Sold Messi to Barca",
		Key:    "notifications.messages.transferSold",
		Params: map[string]any{"player": "Messi", "team": "Barca"},
	}

	cases := map[string]string{
		"en": "en",
		"":   "unset",
		"fr": "unrecognized language",
	}
	for language, label := range cases {
		if got := renderNotificationMessage(msg, language); got != msg.Text {
			t.Errorf("renderNotificationMessage(%s / %s) = %q, want %q", language, label, got, msg.Text)
		}
	}
}

func TestRenderNotificationMessageUnknownKeyFallsBack(t *testing.T) {
	msg := NotificationMessage{
		Text: "Something happened",
		Key:  "notifications.messages.doesNotExist",
	}

	if got := renderNotificationMessage(msg, "es"); got != msg.Text {
		t.Errorf("renderNotificationMessage(unknown key) = %q, want %q", got, msg.Text)
	}
}

func TestRenderNotificationMessageNoKeyFallsBack(t *testing.T) {
	msg := NotificationMessage{Text: "Plain text, no key"}

	if got := renderNotificationMessage(msg, "es"); got != msg.Text {
		t.Errorf("renderNotificationMessage(no key) = %q, want %q", got, msg.Text)
	}
}
