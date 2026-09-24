package model

// PanelEPGSnapshot keeps a complete EPG generation in one atomic MySQL upsert.
// It is separate from the legacy EPGDetails table to preserve operator IDs.
type PanelEPGSnapshot struct {
	ID      uint   `gorm:"primaryKey;autoIncrement:false"`
	Payload []byte `gorm:"type:longblob;not null"`
}
