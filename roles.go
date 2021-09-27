package main

import "strings"

// StudentType is implemented by the different types of student that exist.
type StudentType interface {
	// Codes contains a string array of the different student codes that map to this StudentType.
	Codes() []string

	// Name returns a string representing the name of the student type. It will be singular and lowercase.
	Name() string
}

// Alumnus is a type of student accessible from Active Directory.
type Alumnus struct{}

func (a *Alumnus) Codes() []string {
	return []string{"Alumni"}
}

func (a *Alumnus) Name() string {
	return "alumnus"
}

// CurrentStudent is a type of student accessible from Active Directory.
type CurrentStudent struct{}

func (s *CurrentStudent) Codes() []string {
	return []string{"UG", "PGT", "PGR"}
}

func (a *CurrentStudent) Name() string {
	return "current student"
}

// GetStudentTypeFromCode returns a StudentType corresponding to a given
// code string, or nil if the student type does not exist.
func GetStudentTypeFromCode(code string) StudentType {
	for _, studentType := range []StudentType{&Alumnus{}, &CurrentStudent{}} {
		for _, typeCode := range studentType.Codes() {
			if strings.EqualFold(code, typeCode) {
				return studentType
			}
		}
	}
	return nil
}
