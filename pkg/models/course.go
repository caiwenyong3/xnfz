package models

import "time"

type CourseMode int16

const (
	TeachingMode CourseMode = iota + 1
	PracticeMode
)

type Course struct {
	ID          string
	Name        string
	Description string
	Mode        CourseMode
	Duration    time.Duration // Only applicable for PracticeMode
}
