// form_impl.go — Form CRUD and lifecycle (Create/Get/Update/ListForChapter/
// ListReachable/Publish/Close/Delete). Ported from
// mwanachama-backend-api-gateway's internal/store/memory/survey_store.go and
// survey_store_public.go's Publish half.
package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-forms/gormstore"
	"github.com/aosanya/mwanachama-backend-forms/models"
)

// validateFrame checks the fields Create and Update both write.
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

// Create creates a new Form, always in StatusDraft — any lifecycle stamp the
// caller supplied is discarded.
func (m *formManager) Create(ctx context.Context, f models.Form) (models.Form, error) {
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
	if f.CreatedAt == "" {
		f.CreatedAt = models.NowRFC3339()
	}

	row := gormstore.FormToRow(f)
	if err := m.db.WithContext(ctx).Table(m.tables.Forms).Create(&row).Error; err != nil {
		return models.Form{}, fmt.Errorf("Create: %w", err)
	}
	return gormstore.FormFromRow(row), nil
}

// Get reads a single Form.
func (m *formManager) Get(ctx context.Context, id string) (models.Form, error) {
	var row gormstore.FormRow
	err := m.db.WithContext(ctx).Table(m.tables.Forms).Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Form{}, ErrFormNotFound
		}
		return models.Form{}, fmt.Errorf("Get: %w", err)
	}
	return gormstore.FormFromRow(row), nil
}

// Update rewrites a draft Form's Title/ClosesAt/OpensAt/Audience/
// CollectionMode — every other field (status, every lifecycle stamp) is
// untouched.
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

	now := models.NowRFC3339()
	err = m.db.WithContext(ctx).Table(m.tables.Forms).Where("id = ?", f.ID).
		Updates(map[string]any{
			"title":           f.Title,
			"closes_at":       f.ClosesAt,
			"opens_at":        f.OpensAt,
			"audience":        string(f.Audience),
			"collection_mode": string(f.CollectionMode),
			"updated_at":      now,
		}).Error
	if err != nil {
		return models.Form{}, fmt.Errorf("Update: %w", err)
	}
	cur.Title, cur.ClosesAt, cur.OpensAt = f.Title, f.ClosesAt, f.OpensAt
	cur.Audience, cur.CollectionMode = f.Audience, f.CollectionMode
	cur.LastUpdated = now
	return cur, nil
}

// ListForChapter returns every Form (any status) a chapter originated,
// created_at-then-id order. Ordering by created_at (added to Form purely for
// this) rather than id — see models.Form's doc — since this repo's UUID ids
// carry none of the sequential-insertion-order property the gateway's
// "ORDER BY id" relied on.
func (m *formManager) ListForChapter(ctx context.Context, chapterID string) ([]models.Form, error) {
	var rows []gormstore.FormRow
	if err := m.db.WithContext(ctx).Table(m.tables.Forms).Where("originator_chapter_id = ?", chapterID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListForChapter: %w", err)
	}
	out := make([]models.Form, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.FormFromRow(r))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// ListReachable returns every member-visible Form whose Target set reaches
// ancestry — ancestry[0] is the caller's own chapter, the rest its ancestors
// up to the root. A Target on the caller's own chapter always reaches; a
// Target on an ancestor reaches only when it carries IncludesDescendants.
func (m *formManager) ListReachable(ctx context.Context, ancestry []string) ([]models.Form, error) {
	if len(ancestry) == 0 {
		return []models.Form{}, nil
	}
	own := ancestry[0]
	inChain := make(map[string]bool, len(ancestry))
	for _, id := range ancestry {
		inChain[id] = true
	}

	var targetRows []gormstore.TargetRow
	if err := m.db.WithContext(ctx).Table(m.tables.Targets).Find(&targetRows).Error; err != nil {
		return nil, fmt.Errorf("ListReachable: %w", err)
	}

	seen := map[string]bool{}
	out := []models.Form{}
	for _, tr := range targetRows {
		t := gormstore.TargetFromRow(tr)
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

// Publish opens a draft or approved Form. Idempotent on an already-open
// form (no-op, no error). A public-audience form gets exactly one
// PublicLink (candidateLinkKey stored verbatim — this package neither
// generates nor deduplicates it); a member-audience form gets one
// Propagation per existing Target.
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
		// proceed
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
	updates := map[string]any{
		"status":       string(models.StatusOpen),
		"published_at": now,
		"published_by": publishedBy,
		"updated_at":   now,
	}
	if f.OpensAt == "" {
		updates["opens_at"] = now
		f.OpensAt = now
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Forms).Where("id = ?", id).Updates(updates).Error; err != nil {
		return models.Form{}, models.PublicLink{}, fmt.Errorf("Publish: %w", err)
	}
	f.Status, f.PublishedAt, f.PublishedBy, f.LastUpdated = models.StatusOpen, now, publishedBy, now

	var link models.PublicLink
	if f.Audience == models.AudiencePublic {
		row := gormstore.PublicLinkToRow(models.PublicLink{
			FormID: id, Key: candidateLinkKey, Status: models.LinkActive, CreatedAt: now, CreatedBy: f.CreatedBy,
		})
		if err := m.db.WithContext(ctx).Table(m.tables.PublicLinks).Create(&row).Error; err != nil {
			return models.Form{}, models.PublicLink{}, fmt.Errorf("Publish: %w", err)
		}
		link = gormstore.PublicLinkFromRow(row)
	} else {
		var targetRows []gormstore.TargetRow
		if err := m.db.WithContext(ctx).Table(m.tables.Targets).Where("form_id = ?", id).Find(&targetRows).Error; err != nil {
			return models.Form{}, models.PublicLink{}, fmt.Errorf("Publish: %w", err)
		}
		for _, tr := range targetRows {
			prow := gormstore.PropagationToRow(models.Propagation{
				FormID: id, ChapterID: tr.ChapterID, State: models.PropagationTargeted, TargetedAt: now,
			})
			if err := m.db.WithContext(ctx).Table(m.tables.Propagation).Create(&prow).Error; err != nil {
				return models.Form{}, models.PublicLink{}, fmt.Errorf("Publish: %w", err)
			}
		}
	}
	return f, link, nil
}

// Close closes an open Form. Idempotent on an already-closed form (no-op,
// no error).
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
	err = m.db.WithContext(ctx).Table(m.tables.Forms).Where("id = ?", id).
		Updates(map[string]any{"status": string(models.StatusClosed), "closed_at": now, "closed_by": closedBy, "updated_at": now}).Error
	if err != nil {
		return models.Form{}, fmt.Errorf("Close: %w", err)
	}
	f.Status, f.ClosedAt, f.ClosedBy, f.LastUpdated = models.StatusClosed, now, closedBy, now
	return f, nil
}

// Delete removes a Form and its Questions/QuestionOptions/Targets/Approvals.
// Propagation/PublicLink/Declaration/Answer rows are not swept — by
// construction they exist only once a form is published, and a published
// form is never Deletable, so none can exist here.
func (m *formManager) Delete(ctx context.Context, id string) error {
	f, err := m.Get(ctx, id)
	if err != nil {
		return err
	}
	if !models.Deletable(f.Status) {
		return ErrPublished
	}

	var questionRows []gormstore.QuestionRow
	if err := m.db.WithContext(ctx).Table(m.tables.Questions).Where("form_id = ?", id).Find(&questionRows).Error; err != nil {
		return fmt.Errorf("Delete: %w", err)
	}
	if len(questionRows) > 0 {
		questionIDs := make([]string, len(questionRows))
		for i, q := range questionRows {
			questionIDs[i] = q.ID
		}
		if err := m.db.WithContext(ctx).Table(m.tables.QuestionOptions).Where("question_id IN ?", questionIDs).Delete(&gormstore.QuestionOptionRow{}).Error; err != nil {
			return fmt.Errorf("Delete: %w", err)
		}
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Questions).Where("form_id = ?", id).Delete(&gormstore.QuestionRow{}).Error; err != nil {
		return fmt.Errorf("Delete: %w", err)
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Targets).Where("form_id = ?", id).Delete(&gormstore.TargetRow{}).Error; err != nil {
		return fmt.Errorf("Delete: %w", err)
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Approvals).Where("form_id = ?", id).Delete(&gormstore.ApprovalRow{}).Error; err != nil {
		return fmt.Errorf("Delete: %w", err)
	}
	if err := m.db.WithContext(ctx).Table(m.tables.Forms).Where("id = ?", id).Delete(&gormstore.FormRow{}).Error; err != nil {
		return fmt.Errorf("Delete: %w", err)
	}
	return nil
}
