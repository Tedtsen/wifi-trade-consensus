package events

const (
	BEACON int = iota
	BUY
	REQUEST_VOTE
	REPLY_VOTE
	INFORM_VOTE
	START_FLOW
	TRANSACTION_END
	// Trigger events for simulation of consumer's actions
	TRIGGER_BUY
	// Get stats
	GET_PROVIDER_STATS
)
