package models

import "fmt"

func FileLimit(q Question, defaultMaxFiles int) int {
	if q.MaxFiles != nil {
		return *q.MaxFiles
	}
	return defaultMaxFiles
}

func checkFiles(q Question, a Answer, defaultMaxFiles int) error {
	if len(a.FileIDs) == 0 {
		return fmt.Errorf("%s answer requires at least one file_id", q.AnswerType)
	}
	seen := map[string]bool{}
	for _, id := range a.FileIDs {
		if id == "" {
			return fmt.Errorf("file_id must not be empty")
		}
		if seen[id] {
			return fmt.Errorf("file_id %q given more than once", id)
		}
		seen[id] = true
	}
	if limit := FileLimit(q, defaultMaxFiles); limit > 0 && len(a.FileIDs) > limit {
		return fmt.Errorf("%d file_ids given, more than the limit of %d", len(a.FileIDs), limit)
	}
	if !q.RequireCapture {
		return nil
	}
	captured := map[string]bool{}
	for _, id := range a.CapturedFileIDs {
		captured[id] = true
	}
	for _, id := range a.FileIDs {
		if !captured[id] {
			return fmt.Errorf("file_id %q was not taken now with its time and place", id)
		}
	}
	return nil
}

func validateFileRules(q Question) error {
	if q.MaxFiles != nil && !q.AnswerType.HoldsFiles() {
		return fmt.Errorf("max_files does not apply to a %s question", q.AnswerType)
	}
	if q.RequireCapture && q.AnswerType != AnswerPhoto {
		return fmt.Errorf("require_capture applies only to a photo question")
	}
	if q.MaxFiles != nil && *q.MaxFiles < 1 {
		return fmt.Errorf("max_files must be at least 1")
	}
	return nil
}
