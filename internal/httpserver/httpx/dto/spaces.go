package dto

import "savvy-go/internal/spaces"

// Space is a space as the API lists it, with the caller's role in it.
type Space struct {
	ID   int64  `json:"id"`
	UUID string `json:"uuid"`
	Name string `json:"name"`
	Role string `json:"role"`
}

func NewSpace(s spaces.Space) Space {
	return Space{ID: s.ID, UUID: s.UUID, Name: s.Name, Role: s.Role}
}

// SpaceDetail is the space the request works in, with its limits.
type SpaceDetail struct {
	Space
	Quota int64 `json:"quota"`
	Size  int64 `json:"size"`
}
