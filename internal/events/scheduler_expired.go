package events

// SchedulerExpired is published on scheduler.expired when a scheduled message
// was found far past its deadline and so never fired. The id lets consumers
// reconcile the message; the payload is not republished.
type SchedulerExpired struct {
	ID string `json:"id"`
}

func (SchedulerExpired) Subject() string { return "scheduler.expired" }

func (e SchedulerExpired) MsgID() string { return eventID(e) }
