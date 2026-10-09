package kernel

import "time"

// DomainEvent is something that happened in an aggregate.
type DomainEvent interface {
	EventName() string
	AggregateID() string
	OccurredAt() time.Time
}
