package i18n

import "testing"

func TestResolveAcceptLanguage(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"empty header", "", LocaleFa},
		{"plain fa", "fa", LocaleFa},
		{"plain en", "en", LocaleEn},
		{"uppercase EN", "EN", LocaleEn},
		{"regional subtag fa-IR", "fa-IR", LocaleFa},
		{"regional subtag en-US", "en-US,en;q=0.9", LocaleEn},
		{"q-values prefer fa", "en;q=0.2,fa-IR;q=0.9", LocaleFa},
		{"q-values prefer en", "fa;q=0.3,en-US;q=0.8", LocaleEn},
		{"unsupported only falls back to fa", "de-DE,fr;q=0.5", LocaleFa},
		{"unsupported plus supported", "de,en;q=0.4", LocaleEn},
		{"wildcard falls back to fa", "*;q=1.0", LocaleFa},
		{"malformed falls back to fa", ";;,garbage;;q=", LocaleFa},
		{"default q is 1", "fa;q=0.5,en", LocaleEn},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveAcceptLanguage(tc.header); got != tc.want {
				t.Errorf("ResolveAcceptLanguage(%q) = %q, want %q", tc.header, got, tc.want)
			}
		})
	}
}

func TestMessageLocaleFallbacks(t *testing.T) {
	if got := Message(LocaleEn, "NOT_FOUND"); got != "The requested resource was not found." {
		t.Errorf("en NOT_FOUND = %q", got)
	}
	if got := Message(LocaleFa, "NOT_FOUND"); got != "منبع درخواست‌شده پیدا نشد." {
		t.Errorf("fa NOT_FOUND = %q", got)
	}
	// Unsupported locale falls back to fa.
	if got := Message("de", "NOT_FOUND"); got != Message(LocaleFa, "NOT_FOUND") {
		t.Errorf("de NOT_FOUND = %q, want the fa fallback", got)
	}
	// Unknown code degrades to the code itself.
	if got := Message(LocaleFa, "NOPE"); got != "NOPE" {
		t.Errorf("unknown code = %q, want NOPE", got)
	}
}
