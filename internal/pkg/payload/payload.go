package payload

import "github.com/google/uuid"

type Meta struct {
	PayloadType   int       `json:"type"`
	TransactionID uuid.UUID `json:"transaction_id"`
	OriginID      uuid.UUID `json:"origin_id"`
	OriginAddress string    `json:"origin_address"`
	// Size                  int       `json:"size"`
	// Utilization           int       `json:"utilization"`
}
