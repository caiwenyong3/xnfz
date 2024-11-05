package models

import "time"

type CourseMode int

const (
	TeachingMode CourseMode = iota
	PracticeMode
)

type Course struct {
	ID          string
	Name        string
	Description string
	Mode        CourseMode
	Duration    time.Duration // Only applicable for PracticeMode
}
