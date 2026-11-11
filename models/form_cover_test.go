package models

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestOnlyUnpublishedFormsAreDeletable(t *testing.T) {
	want := map[Status]bool{StatusDraft: true, StatusSubmitted: true, StatusApproved: true, StatusOpen: false, StatusClosed: false}
	for s, deletable := range want {
		if Deletable(s) != deletable {
			t.Fatalf("Deletable(%s) = %v, want %v", s, !deletable, deletable)
		}
		if VisibleToMembers(s) == deletable {
			t.Fatalf("VisibleToMembers(%s) should be the opposite of Deletable", s)
		}
	}
}

func TestValidateWindow(t *testing.T) {
	cases := []struct {
		name, opens, closes, refusal string
	}{
		{"no closing time", "2026-01-01T00:00:00Z", "", ""},
		{"closing only", "", "2026-01-02T00:00:00Z", ""},
		{"closes after opens", "2026-01-01T00:00:00Z", "2026-01-01T00:00:01Z", ""},
		{"closes equals opens", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z", "closes_at must be after opens_at"},
		{"closes before opens", "2026-01-02T00:00:00Z", "2026-01-01T00:00:00Z", "closes_at must be after opens_at"},
		{"bad closes", "", "tomorrow", "closes_at:"},
		{"bad opens", "yesterday", "2026-01-01T00:00:00Z", "opens_at:"},
		{"offsets compare as instants", "2026-01-01T02:00:00+02:00", "2026-01-01T00:30:00Z", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateWindow(c.opens, c.closes)
			if c.refusal == "" && err != nil {
				t.Fatalf("want accepted, got %v", err)
			}
			if c.refusal != "" && (err == nil || !strings.Contains(err.Error(), c.refusal)) {
				t.Fatalf("want %q, got %v", c.refusal, err)
			}
		})
	}
}

func TestAFormKeepsItsWindowUnlessAdvisory(t *testing.T) {
	opens := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	closes := opens.Add(24 * time.Hour)
	form := Form{OpensAt: opens.Format(time.RFC3339Nano), ClosesAt: closes.Format(time.RFC3339Nano)}

	cases := []struct {
		name        string
		now         time.Time
		outside     bool
		pastClosing bool
	}{
		{"before opening", opens.Add(-time.Second), true, false},
		{"inside", opens.Add(time.Hour), false, false},
		{"after closing", closes.Add(time.Second), true, true},
	}
	for _, c := range cases {
		if got := form.OutsideWindow(c.now); got != c.outside {
			t.Fatalf("%s: OutsideWindow = %v, want %v", c.name, got, c.outside)
		}
		if got := form.PastClosing(c.now); got != c.pastClosing {
			t.Fatalf("%s: PastClosing = %v, want %v", c.name, got, c.pastClosing)
		}
	}

	advisory := form
	advisory.WindowAdvisory = true
	for _, c := range cases {
		if advisory.OutsideWindow(c.now) || advisory.PastClosing(c.now) {
			t.Fatalf("%s: an advisory window must never shut a form", c.name)
		}
	}

	unbounded := Form{}
	if unbounded.OutsideWindow(closes.Add(1000*time.Hour)) || unbounded.PastClosing(closes) {
		t.Fatal("a form with no window is never outside it")
	}
}

func TestAMemberAudienceFormCannotBeInterviewed(t *testing.T) {
	if err := ValidateAudienceCollection(AudienceMember, CollectionInterviewer); err == nil {
		t.Fatal("member audience with interviewer collection must be refused")
	}
	for _, ok := range [][2]string{{"member", "self"}, {"public", "self"}, {"public", "interviewer"}} {
		if err := ValidateAudienceCollection(Audience(ok[0]), CollectionMode(ok[1])); err != nil {
			t.Fatalf("%v should be accepted: %v", ok, err)
		}
	}
}

func TestPublishingAuthorityIsOneLineOfBoundedLength(t *testing.T) {
	if err := ValidatePublishingAuthority(""); err != nil {
		t.Fatalf("empty is fine: %v", err)
	}
	if err := ValidatePublishingAuthority(strings.Repeat("é", MaxPublishingAuthority)); err != nil {
		t.Fatalf("exactly the limit in runes is fine: %v", err)
	}
	if err := ValidatePublishingAuthority(strings.Repeat("é", MaxPublishingAuthority+1)); err == nil {
		t.Fatal("one rune over the limit must be refused")
	}
	for _, bad := range []string{"two\nlines", "tab\there", "nul\x00"} {
		if err := ValidatePublishingAuthority(bad); err == nil || !strings.Contains(err.Error(), "one line") {
			t.Fatalf("%q should be refused as multi-line, got %v", bad, err)
		}
	}
}

func TestShortenPublisherName(t *testing.T) {
	if got := ShortenPublisherName("  Ada\nLovelace\t "); got != "Ada Lovelace" {
		t.Fatalf("got %q", got)
	}
	long := strings.Repeat("ñ", MaxPublisherName+30)
	got := ShortenPublisherName(long)
	if utf8.RuneCountInString(got) != MaxPublisherName || !utf8.ValidString(got) {
		t.Fatalf("got %d runes valid=%v, want %d valid runes", utf8.RuneCountInString(got), utf8.ValidString(got), MaxPublisherName)
	}
	cut := strings.Repeat("a", MaxPublisherName-1) + " b"
	if got := ShortenPublisherName(cut); got != strings.Repeat("a", MaxPublisherName-1) {
		t.Fatalf("trailing space left by the cut must be trimmed, got %d runes", utf8.RuneCountInString(got))
	}
}

func TestParseFormTypes(t *testing.T) {
	good := []any{
		map[string]any{"name": "survey", "audience": "member", "collection_mode": "self"},
		map[string]any{"name": "census", "audience": "public", "collection_mode": "interviewer"},
	}
	types, err := ParseFormTypes(good)
	if err != nil || len(types) != 2 || types[1].Name != "census" || types[1].CollectionMode != CollectionInterviewer {
		t.Fatalf("got %+v, %v", types, err)
	}
	empty, err := ParseFormTypes(nil)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("nil should give an empty non-nil list, got %#v, %v", empty, err)
	}

	cases := []struct {
		name    string
		value   any
		refusal string
	}{
		{"not a list", "survey", "form_types:"},
		{"no name", []any{map[string]any{"audience": "public", "collection_mode": "self"}}, "has no name"},
		{"duplicate name", []any{
			map[string]any{"name": "a", "audience": "public", "collection_mode": "self"},
			map[string]any{"name": "a", "audience": "public", "collection_mode": "self"},
		}, `"a" is named twice`},
		{"bad audience", []any{map[string]any{"name": "a", "audience": "staff", "collection_mode": "self"}}, `audience "staff"`},
		{"bad mode", []any{map[string]any{"name": "a", "audience": "public", "collection_mode": "phone"}}, `collection_mode "phone"`},
		{"member interviewed", []any{map[string]any{"name": "a", "audience": "member", "collection_mode": "interviewer"}}, "cannot use interviewer collection"},
		{"unmarshalable", make(chan int), "form_types:"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseFormTypes(c.value)
			if err == nil || !strings.Contains(err.Error(), c.refusal) {
				t.Fatalf("want %q, got %v", c.refusal, err)
			}
		})
	}
}
