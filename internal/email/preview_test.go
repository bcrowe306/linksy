package email

import (
	"os"
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	c := Content{Heading: "New sign-in <script>", Paragraphs: []string{"Hello & welcome"}, Details: [][2]string{{"IP", "1.2.3.4"}},
		Button: &Button{Label: "Review", URL: "https://lynqr.app/profile?tab=security&x=1"}, Reason: "Because", ManageURL: "https://lynqr.app/m", Unsubscribe: "https://lynqr.app/u"}
	h, txt := c.Render()
	if strings.Contains(h, "<script>") || !strings.Contains(h, "&lt;script&gt;") || !strings.Contains(h, "tab=security&amp;x=1") {
		t.Fatal("HTML not escaped")
	}
	if !strings.Contains(txt, "Review: https://lynqr.app/profile?tab=security&x=1") || !strings.Contains(txt, "Unsubscribe: https://lynqr.app/u") {
		t.Fatalf("text body: %s", txt)
	}
	if p := os.Getenv("EMAIL_PREVIEW"); p != "" {
		_ = os.WriteFile(p, []byte(h), 0o644)
	}
}
