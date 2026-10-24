package models

type Send struct {
	Submission Submission `json:"submission"`
	Answers    []Answer   `json:"answers"`
}
