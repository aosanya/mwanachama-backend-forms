package mwanachamaforms

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"unicode/utf8"

	"github.com/aosanya/mwanachama-backend-forms/models"
)

const MaxReportTextBytes = 200000

type ReportTextReader func(ctx context.Context, gitPath, gitCommit string) (string, error)

func (m *formManager) UseReportText(r ReportTextReader) { m.reportText = r }

func (m *formManager) ListMyResults(ctx context.Context, callerID string) ([]models.PublishedReport, error) {
	out := []models.PublishedReport{}
	if callerID == "" || m.groups == nil {
		return out, nil
	}
	groups, err := m.groups(ctx, callerID)
	if err != nil {
		return nil, fmt.Errorf("ListMyResults: %w", err)
	}
	reports, err := listOf[models.ResultReport](m, m.q(ctx, roleResultReport).Order("version desc"), roleResultReport)
	if err != nil {
		return nil, fmt.Errorf("ListMyResults: %w", err)
	}
	seen := map[string]bool{}
	for _, report := range reports {
		if seen[report.FormID] {
			continue
		}
		seen[report.FormID] = true
		reaches, err := m.reportReachesCaller(ctx, report, callerID, groups)
		if err != nil {
			return nil, fmt.Errorf("ListMyResults: %w", err)
		}
		if !reaches {
			continue
		}
		f, err := m.Get(unnarrowed(ctx), report.FormID)
		if errors.Is(err, ErrFormNotFound) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("ListMyResults: %w", err)
		}
		out = append(out, models.PublishedReport{
			FormID: report.FormID, FormTitle: f.Title, Version: report.Version,
			PublishedAt: report.PublishedAt, GitPath: report.GitPath,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PublishedAt != out[j].PublishedAt {
			return out[i].PublishedAt > out[j].PublishedAt
		}
		return out[i].FormID < out[j].FormID
	})
	return out, nil
}

func (m *formManager) ReadResultText(ctx context.Context, formID, callerID string) (models.ReportText, error) {
	report, err := m.LatestResults(ctx, formID, callerID)
	if err != nil {
		return models.ReportText{}, err
	}
	if m.reportText == nil {
		return models.ReportText{}, ErrReportTextUnavailable
	}
	f, err := m.Get(unnarrowed(ctx), formID)
	if errors.Is(err, ErrFormNotFound) {
		return models.ReportText{}, ErrReportNotFound
	}
	if err != nil {
		return models.ReportText{}, fmt.Errorf("ReadResultText: %w", err)
	}
	whole, err := m.reportText(ctx, report.GitPath, report.GitCommit)
	if err != nil {
		slog.WarnContext(ctx, "a report's text could not be read", "form_id", formID, "git_path", report.GitPath, "error", err)
		return models.ReportText{}, ErrReportTextUnavailable
	}
	if !utf8.ValidString(whole) {
		return models.ReportText{}, ErrReportNotText
	}
	text := cutAtCharacter(whole, MaxReportTextBytes)
	return models.ReportText{
		FormID: formID, FormTitle: f.Title, Version: report.Version, PublishedAt: report.PublishedAt,
		GitPath: report.GitPath, Text: text, Size: len(whole), Truncated: len(text) < len(whole),
	}, nil
}

func cutAtCharacter(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}
