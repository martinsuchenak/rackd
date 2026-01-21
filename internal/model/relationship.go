package model

import "time"

type DeviceRelationship struct {
	ParentID  string    `json:"parent_id"`
	ChildID   string    `json:"child_id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

const (
	RelationshipContains    = "contains"
	RelationshipConnectedTo = "connected_to"
	RelationshipDependsOn   = "depends_on"
)
