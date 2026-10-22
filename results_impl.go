package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

func (m *formManager) PublishResults(ctx context.Context, formID, gitPath, gitCommit, publishedBy string) (models.ResultReport, error) {
	gitPath, gitCommit = strings.TrimSpace(gitPath), strings.TrimSpace(gitCommit)
	if gitPath == "" || gitCommit == "" || publishedBy == "" {
		return models.ResultReport{}, ErrInvalidReport
	}
	f, err := m.Get(unnarrowed(ctx), formID)
	if err != nil {
		return models.ResultReport{}, err
	}
	if f.Status != models.StatusClosed {
		return models.ResultReport{}, ErrNotClosed
	}
	targets, err := m.ListTargets(ctx, formID)
	if err != nil {
		return models.ResultReport{}, fmt.Errorf("PublishResults: %w", err)
	}
	if len(targets) == 0 {
		return models.ResultReport{}, ErrNoTarget
	}
	now := models.NowRFC3339()
	report := models.ResultReport{
		ID: newID(), FormID: formID, ChapterID: targets[0].ChapterID,
		GitPath: gitPath, GitCommit: gitCommit, PublishedBy: publishedBy, PublishedAt: now,
	}
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var held int64
		if err := tx.Table(m.table(roleResultReport)).Where("form_id = ?", formID).Count(&held).Error; err != nil {
			return err
		}
		report.Version = int(held) + 1
		if err := m.insertTx(tx, roleResultReport, report); err != nil {
			return err
		}
		f.ResultsPublishedAt, f.LastUpdated = now, now
		return m.replaceTx(tx, roleForm, formID, f)
	})
	if err != nil {
		return models.ResultReport{}, fmt.Errorf("PublishResults: %w", err)
	}
	return report, nil
}

func (m *formManager) LatestResults(ctx context.Context, formID, callerID string) (models.ResultReport, error) {
	if callerID == "" || m.groups == nil {
		return models.ResultReport{}, ErrReportNotFound
	}
	var report models.ResultReport
	q := m.q(ctx, roleResultReport).Where("form_id = ?", formID).Order("version desc")
	if err := m.take(q, roleResultReport, &report, ErrReportNotFound); err != nil {
		if errors.Is(err, ErrReportNotFound) {
			return models.ResultReport{}, err
		}
		return models.ResultReport{}, fmt.Errorf("LatestResults: %w", err)
	}
	groups, err := m.groups(ctx, callerID)
	if err != nil {
		return models.ResultReport{}, fmt.Errorf("LatestResults: %w", err)
	}
	reaches, err := m.reportReaches(ctx, report, groups)
	if err != nil {
		return models.ResultReport{}, fmt.Errorf("LatestResults: %w", err)
	}
	if !reaches {
		return models.ResultReport{}, ErrReportNotFound
	}
	return report, nil
}

func (m *formManager) reportReaches(ctx context.Context, report models.ResultReport, groups []string) (bool, error) {
	if slices.Contains(groups, report.ChapterID) {
		return true, nil
	}
	targets, err := m.ListTargets(unnarrowed(ctx), report.FormID)
	if err != nil {
		return false, err
	}
	for _, t := range targets {
		if slices.Contains(groups, t.ChapterID) {
			return true, nil
		}
	}
	return false, nil
}
