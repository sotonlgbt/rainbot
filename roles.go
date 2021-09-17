package main

type StudentType interface {
	codes() []string
}

type Alumnus struct{}

func (a *Alumnus) codes() []string {
	return []string{"Alumni"}
}

type CurrentStudent struct{}

func (s *CurrentStudent) codes() []string {
	return []string{"UG", "PGT", "PGR"}
}
