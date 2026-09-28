package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func validateFrame(f models.Form) error {
	if f.Title == "" {
		return ErrMissingTitle
	}
	if f.ClosesAt == "" {
		return ErrMissingClosesAt
	}
	if err := models.ValidateWindow(f.OpensAt, f.ClosesAt); err != nil {
		return fmt.Errorf("%w: %v", ErrWindowInvalid, err)
	}
	return nil
}

func (m *formManager) Create(ctx context.Context, f models.Form) (models.Form, error) {
	f.ID = ""
	if err := validateFrame(f); err != nil {
		return models.Form{}, err
	}
	closesAt, _ := time.Parse(time.RFC3339Nano, f.ClosesAt)
	if !closesAt.After(time.Now()) {
		return models.Form{}, ErrClosesInPast
	}
	if f.Audience == "" {
		f.Audience = models.AudienceMember
	}
	if f.CollectionMode == "" {
		f.CollectionMode = models.CollectionSelf
	}
	if err := models.ValidateAudienceCollection(f.Audience, f.CollectionMode); err != nil {
		return models.Form{}, fmt.Errorf("%w: %v", ErrAudienceConflict, err)
	}

	f.Status = models.StatusDraft
	f.PublishedAt, f.PublishedBy = "", ""
	f.ClosedAt, f.ClosedBy = "", ""
	f.SubmittedAt, f.SubmittedBy = "", ""
	f.ApprovedAt, f.ApprovedBy = "", ""
	f.ResultsPublishedAt = ""

	now := models.NowRFC3339()
	f.ID = newID()
	f.CreatedAt, f.LastUpdated = now, now
	if err := m.checks(roleForm, f); err != nil {
		return models.Form{}, err
	}
	if err := m.insert(ctx, roleForm, f); err != nil {
		return models.Form{}, fmt.Errorf("Create: %w", err)
	}
	return f, nil
}

func (m *formManager) Get(ctx context.Context, id string) (models.Form, error) {
	var f models.Form
	if err := m.find(ctx, roleForm, id, &f, ErrFormNotFound); err != nil {
		if errors.Is(err, ErrFormNotFound) {
			return models.Form{}, err
		}
		return models.Form{}, fmt.Errorf("Get: %w", err)
	}
	return f, nil
}

func (m *formManager) Update(ctx context.Context, f models.Form) (models.Form, error) {
	cur, err := m.Get(ctx, f.ID)
	if err != nil {
		return models.Form{}, err
	}
	if cur.Status != models.StatusDraft {
		return models.Form{}, ErrNotDraft
	}
	if err := validateFrame(f); err != nil {
		return models.Form{}, err
	}
	if err := models.ValidateAudienceCollection(f.Audience, f.CollectionMode); err != nil {
		return models.Form{}, fmt.Errorf("%w: %v", ErrAudienceConflict, err)
	}

	cur.Title, cur.ClosesAt, cur.OpensAt = f.Title, f.ClosesAt, f.OpensAt
	cur.Audience, cur.CollectionMode = f.Audience, f.CollectionMode
	cur.LastUpdated = models.NowRFC3339()
	if err := m.checks(roleForm, cur); err != nil {
		return models.Form{}, err
	}
	if err := m.replace(ctx, roleForm, cur.ID, cur); err != nil {
		return models.Form{}, fmt.Errorf("Update: %w", err)
	}
	return cur, nil
}

func (m *formManager) ListForChapter(ctx context.Context, chapterID string) ([]models.Form, error) {
	out, err := listOf[models.Form](m, m.q(ctx, roleForm).Where("originator_chapter_id = ?", chapterID), roleForm)
	if err != nil {
		return nil, fmt.Errorf("ListForChapter: %w", err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (m *formManager) ListReachable(ctx context.Context, ancestry []string) ([]models.Form, error) {
	if len(ancestry) == 0 {
		return []models.Form{}, nil
	}
	own := ancestry[0]
	inChain := make(map[string]bool, len(ancestry))
	for _, id := range ancestry {
		inChain[id] = true
	}

	targets, err := listOf[models.Target](m, m.q(ctx, roleTarget), roleTarget)
	if err != nil {
		return nil, fmt.Errorf("ListReachable: %w", err)
	}

	seen := map[string]bool{}
	out := []models.Form{}
	for _, t := range targets {
		if !inChain[t.ChapterID] {
			continue
		}
		if t.ChapterID != own && !t.IncludesDescendants {
			continue
		}
		if seen[t.FormID] {
			continue
		}
		f, err := m.Get(ctx, t.FormID)
		if err != nil {
			if errors.Is(err, ErrFormNotFound) {
				continue
			}
			return nil, fmt.Errorf("ListReachable: %w", err)
		}
		if !models.VisibleToMembers(f.Status) {
			continue
		}
		seen[t.FormID] = true
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *formManager) Publish(ctx context.Context, id, candidateLinkKey, publishedBy string) (models.Form, models.PublicLink, error) {
	f, err := m.Get(ctx, id)
	if err != nil {
		return models.Form{}, models.PublicLink{}, err
	}
	switch f.Status {
	case models.StatusOpen:
		return f, models.PublicLink{}, nil
	case models.StatusSubmitted:
		return models.Form{}, models.PublicLink{}, ErrAwaitingApproval
	case models.StatusDraft, models.StatusApproved:
	default:
		return models.Form{}, models.PublicLink{}, ErrNotDraft
	}
	closesAt, err := time.Parse(time.RFC3339Nano, f.ClosesAt)
	if err != nil {
		return models.Form{}, models.PublicLink{}, fmt.Errorf("Publish: closes_at: %w", err)
	}
	if !closesAt.After(time.Now()) {
		return models.Form{}, models.PublicLink{}, ErrClosesInPast
	}

	now := models.NowRFC3339()
	if f.OpensAt == "" {
		f.OpensAt = now
	}
	f.Status, f.PublishedAt, f.PublishedBy, f.LastUpdated = models.StatusOpen, now, publishedBy, now

	// One transaction: a failed link or propagation insert must roll the
	// status change back, or the Form is left open with no way to publish it
	// again.
	var link models.PublicLink
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := encode(m.st.Object(roleForm), f)
		if err != nil {
			return err
		}
		if err := tx.Table(m.table(roleForm)).Where("id = ?", id).Updates(row).Error; err != nil {
			return err
		}
		if f.Audience == models.AudiencePublic {
			link, err = m.mintLink(tx, f, candidateLinkKey, now)
			return err
		}
		return m.spreadToTargets(tx, id, now)
	})
	if err != nil {
		if errors.Is(err, ErrLinkKeyTaken) {
			return models.Form{}, models.PublicLink{}, err
		}
		return models.Form{}, models.PublicLink{}, fmt.Errorf("Publish: %w", err)
	}
	return f, link, nil
}

func (m *formManager) mintLink(tx *gorm.DB, f models.Form, key, now string) (models.PublicLink, error) {
	var taken int64
	if err := tx.Table(m.table(rolePublicLink)).Where("key = ?", key).Count(&taken).Error; err != nil {
		return models.PublicLink{}, err
	}
	if taken > 0 {
		return models.PublicLink{}, ErrLinkKeyTaken
	}
	link := models.PublicLink{
		ID: newID(), FormID: f.ID, Key: key, Status: models.LinkActive,
		CreatedAt: now, CreatedBy: f.CreatedBy,
	}
	return link, m.insertTx(tx, rolePublicLink, link)
}

func (m *formManager) spreadToTargets(tx *gorm.DB, formID, now string) error {
	targets, err := listOf[models.Target](m, tx.Table(m.table(roleTarget)).Where("form_id = ?", formID), roleTarget)
	if err != nil {
		return err
	}
	for _, t := range targets {
		p := models.Propagation{
			ID: newID(), FormID: formID, ChapterID: t.ChapterID,
			State: models.PropagationTargeted, TargetedAt: now,
		}
		if err := m.insertTx(tx, rolePropagation, p); err != nil {
			return err
		}
	}
	return nil
}

func (m *formManager) Close(ctx context.Context, id, closedBy string) (models.Form, error) {
	f, err := m.Get(ctx, id)
	if err != nil {
		return models.Form{}, err
	}
	if f.Status == models.StatusClosed {
		return f, nil
	}
	if f.Status != models.StatusOpen {
		return models.Form{}, ErrNotOpen
	}
	now := models.NowRFC3339()
	f.Status, f.ClosedAt, f.ClosedBy, f.LastUpdated = models.StatusClosed, now, closedBy, now
	if err := m.replace(ctx, roleForm, id, f); err != nil {
		return models.Form{}, fmt.Errorf("Close: %w", err)
	}
	return f, nil
}

// Delete removes a Form and its Questions/QuestionOptions/Targets/Approvals.
// Propagation, PublicLink, Declaration and Answer rows are not swept — by
// construction they exist only once a form is published, and a published
// form is never deletable, so none can be here to sweep.
func (m *formManager) Delete(ctx context.Context, id string) error {
	f, err := m.Get(ctx, id)
	if err != nil {
		return err
	}
	if !models.Deletable(f.Status) {
		return ErrPublished
	}

	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var questionIDs []string
		if err := tx.Table(m.table(roleQuestion)).Where("form_id = ?", id).Pluck("id", &questionIDs).Error; err != nil {
			return err
		}
		if len(questionIDs) > 0 {
			if err := m.deleteWhere(tx, roleQuestionOption, "question_id IN ?", questionIDs); err != nil {
				return err
			}
		}
		for _, role := range []string{roleQuestion, roleTarget, roleApproval} {
			if err := m.deleteWhere(tx, role, "form_id = ?", id); err != nil {
				return err
			}
		}
		return m.deleteWhere(tx, roleForm, "id = ?", id)
	})
	if err != nil {
		return fmt.Errorf("Delete: %w", err)
	}
	return nil
}

func (m *formManager) deleteWhere(tx *gorm.DB, role, where string, args ...any) error {
	return tx.Table(m.table(role)).Where(where, args...).Delete(nil).Error
}
