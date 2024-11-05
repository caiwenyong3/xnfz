package models

type UserRole int

const (
	Teacher UserRole = iota
	Student
	Observer
)

type User struct {
	ID   string
	Name string
	Role UserRole
}
