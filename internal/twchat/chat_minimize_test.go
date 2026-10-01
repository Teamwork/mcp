package twchat

import (
	"strings"
	"testing"
)

// TestMinimizeChatBodyRestrictsPeople pins that, for a restricted
// installation, a person keeps who they are and loses presence, avatar,
// timezone and activity, wherever the person appears, while a conversation's
// own status survives.
func TestMinimizeChatBodyRestrictsPeople(t *testing.T) {
	body := `{"people":[{"id":777,"firstName":"John","status":"away","avatar":"a.jpg",` +
		`"timezoneReferenceCode":"Europe/Dublin","roomId":12345}],` +
		`"conversations":[{"id":1,"status":"active","people":[{"id":777,"lastActivityAt":"2026-01-01T00:00:00Z"}],` +
		`"latestMessage":{"author":{"id":777,"fullName":"John Doe","status":"active","avatar":"a.jpg"}}}],` +
		`"account":{"id":777,"subscription":{"pricePlanId":103},"user":{"id":777,"status":"online"}}}`

	got, err := minimizeChatBody([]byte(body), true)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	for _, s := range []string{"away", "avatar", "timezoneReferenceCode", "roomId", "lastActivityAt", "online",
		"subscription"} {
		if strings.Contains(text, s) {
			t.Errorf("%s reached the caller: %s", s, text)
		}
	}
	for _, s := range []string{`"firstName":"John"`, `"fullName":"John Doe"`, `"id":1,"latestMessage"`} {
		if !strings.Contains(text, s) {
			t.Errorf("expected %s in %s", s, text)
		}
	}
	if !strings.Contains(text, `"status":"active"}`) && !strings.Contains(text, `"status":"active",`) {
		t.Errorf("the conversation's own status was dropped: %s", text)
	}

	unrestricted, err := minimizeChatBody([]byte(body), false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(unrestricted), `"status":"away"`) ||
		!strings.Contains(string(unrestricted), `"subscription"`) {
		t.Errorf("an unrestricted body lost a person attribute: %s", unrestricted)
	}
}
