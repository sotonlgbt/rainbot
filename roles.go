package main

// StudentType is implemented by the different types of student that exist.
type StudentType interface {
	// Codes contains a string array of the different student codes that map to this StudentType.
	Codes() []string
}

// Alumnus is a type of student accessible from Active Directory.
type Alumnus struct{}

func (a *Alumnus) Codes() []string {
	return []string{"Alumni"}
}

// CurrentStudent is a type of student accessible from Active Directory.
type CurrentStudent struct{}

func (s *CurrentStudent) Codes() []string {
	return []string{"UG", "PGT", "PGR"}
}
