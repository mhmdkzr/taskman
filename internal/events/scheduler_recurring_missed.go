package events

// RecurringMissed is published on scheduler.recurring.missed when an occurrence
// of a recurring schedule was found far past its deadline and so was skipped:
// it never fired and the schedule advanced past it. RecurrenceID names the
// schedule; OccurredAt is the occurrence's scheduled publish time (unix ms),
// so consumers can tell which period was missed.
type RecurringMissed struct {
	RecurrenceID string `json:"recurrence_id"`
	OccurredAt   int64  `json:"occurred_at"`
}

func (RecurringMissed) Subject() string { return "scheduler.recurring.missed" }

func (e RecurringMissed) MsgID() string { return eventID(e) }
