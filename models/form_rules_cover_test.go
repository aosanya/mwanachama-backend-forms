package models

import (
	"strings"
	"testing"
	"time"
)

func TestOnlyDraftsSubmittedAndApprovedFormsCanBeDeleted(t *testing.T) {
	for status, visible := range map[Status]bool{
		StatusDraft: false, StatusSubmitted: false, StatusApproved: false, StatusOpen: true, StatusClosed: true,
	} {
		if VisibleToMembers(status) != visible {
			t.Errorf("VisibleToMembers(%s) = %v, want %v", status, !visible, visible)
		}
		if Deletable(status) == visible {
			t.Errorf("Deletable(%s) = %v, want %v", status, Deletable(status), !visible)
		}
	}
}

func TestValidateWindowNeedsAClosingAfterTheOpening(t *testing.T) {
	cases := []struct {
		name         string
		opens, close string
		wantErr      string
	}{
		{"no closing", "2026-11-10T08:00:00Z", "", ""},
		{"closing only", "", "2026-11-12T08:00:00Z", ""},
		{"closing after opening", "2026-11-10T08:00:00Z", "2026-11-12T08:00:00Z", ""},
		{"closing at the opening", "2026-11-10T08:00:00Z", "2026-11-10T08:00:00Z", "closes_at must be after opens_at"},
		{"closing before the opening", "2026-11-10T08:00:00Z", "2026-11-09T08:00:00Z", "closes_at must be after opens_at"},
		{"closing is not a time", "2026-11-10T08:00:00Z", "tomorrow", "closes_at"},
		{"opening is not a time", "soon", "2026-11-12T08:00:00Z", "opens_at"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateWindow(c.opens, c.close)
			switch {
			case c.wantErr == "" && err != nil:
				t.Fatalf("refused a good window: %v", err)
			case c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)):
				t.Fatalf("error = %v, want one naming %q", err, c.wantErr)
			}
		})
	}
}

func TestAFormIsOutsideItsWindowBeforeOpeningAndAfterClosing(t *testing.T) {
	f := Form{OpensAt: "2026-11-10T08:00:00Z", ClosesAt: "2026-11-12T08:00:00Z"}
	at := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	if !f.OutsideWindow(at("2026-11-10T07:59:59Z")) {
		t.Error("a form must be closed to answers before it opens")
	}
	if f.OutsideWindow(at("2026-11-11T08:00:00Z")) {
		t.Error("a form inside its window is not outside it")
	}
	if !f.OutsideWindow(at("2026-11-12T08:00:01Z")) || !f.PastClosing(at("2026-11-12T08:00:01Z")) {
		t.Error("a form must be closed to answers after it closes")
	}
	if f.PastClosing(at("2026-11-10T07:00:00Z")) {
		t.Error("a form that has not opened is not past its closing")
	}

	advisory := f
	advisory.WindowAdvisory = true
	if advisory.OutsideWindow(at("2030-01-01T00:00:00Z")) || advisory.PastClosing(at("2030-01-01T00:00:00Z")) {
		t.Error("an advisory window never shuts anyone out")
	}

	if (Form{}).OutsideWindow(at("2030-01-01T00:00:00Z")) {
		t.Error("a form with no window is never outside it")
	}
}

func TestLandingNeedsAStandingForm(t *testing.T) {
	if err := ValidateLanding(false, LandingNone); err != nil {
		t.Fatalf("no landing is fine on any form: %v", err)
	}
	if err := ValidateLanding(true, LandingCase); err != nil {
		t.Fatalf("a standing form may land: %v", err)
	}
	err := ValidateLanding(false, LandingThread)
	if err == nil || !strings.Contains(err.Error(), "standing") {
		t.Fatalf("landing on a form that is not standing = %v", err)
	}
}

func TestOneVoiceOnlyFitsAMemberAudienceFormThatIsNotStanding(t *testing.T) {
	if err := ValidateOneVoice(OneVoiceOff, true, AudiencePublic); err != nil {
		t.Fatalf("one voice off is always fine: %v", err)
	}
	for _, v := range []OneVoice{OneVoiceRefuse, OneVoiceFlag} {
		if err := ValidateOneVoice(v, false, AudienceMember); err != nil {
			t.Fatalf("%s on a member form that is not standing: %v", v, err)
		}
	}
	if err := ValidateOneVoice(OneVoiceRefuse, true, AudienceMember); err == nil {
		t.Fatal("one voice on a standing form was accepted")
	}
	if err := ValidateOneVoice(OneVoiceFlag, false, AudiencePublic); err == nil {
		t.Fatal("one voice on a public form was accepted")
	}
	if err := ValidateOneVoice(OneVoice("shout"), false, AudienceMember); err == nil || !strings.Contains(err.Error(), "shout") {
		t.Fatalf("an unknown one voice value = %v, want an error naming it", err)
	}
}

func TestAMemberFormCannotCollectThroughAnInterviewer(t *testing.T) {
	if err := ValidateAudienceCollection(AudienceMember, CollectionInterviewer); err == nil {
		t.Fatal("member audience with interviewer collection was accepted")
	}
	for _, c := range []struct {
		a Audience
		m CollectionMode
	}{{AudienceMember, CollectionSelf}, {AudiencePublic, CollectionSelf}, {AudiencePublic, CollectionInterviewer}} {
		if err := ValidateAudienceCollection(c.a, c.m); err != nil {
			t.Errorf("%s with %s was refused: %v", c.a, c.m, err)
		}
	}
}

func TestPublishingAuthorityIsOneShortLineOfText(t *testing.T) {
	if err := ValidatePublishingAuthority("Ward office, Nairobi"); err != nil {
		t.Fatalf("a plain authority was refused: %v", err)
	}
	if err := ValidatePublishingAuthority(strings.Repeat("é", MaxPublishingAuthority)); err != nil {
		t.Fatalf("an authority of exactly the limit, counted in characters, was refused: %v", err)
	}
	if err := ValidatePublishingAuthority(strings.Repeat("é", MaxPublishingAuthority+1)); err == nil {
		t.Fatal("an authority over the limit was accepted")
	}
	if err := ValidatePublishingAuthority("two\nlines"); err == nil {
		t.Fatal("an authority with a line break was accepted")
	}
}

func TestAPublisherNameIsTrimmedFlattenedAndShortened(t *testing.T) {
	if got := ShortenPublisherName("  Ward\toffice\n"); got != "Ward office" {
		t.Fatalf("ShortenPublisherName = %q", got)
	}
	long := strings.Repeat("ü", MaxPublisherName+30)
	got := ShortenPublisherName(long)
	if len([]rune(got)) != MaxPublisherName {
		t.Fatalf("a long name must be cut to %d characters, got %d", MaxPublisherName, len([]rune(got)))
	}
	if ShortenPublisherName("") != "" {
		t.Fatal("an empty name stays empty")
	}
}
