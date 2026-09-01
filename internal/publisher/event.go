package publisher

// Event is any value with a subject to publish it on and a deterministic
// dedup id. Event definitions live in the events package; anything that
// satisfies the interface is publishable. MsgID must be derived from the
// event's contents so that re-publishing the same event always yields the
// same id: the publisher uses it to deduplicate within the stream's dedup
// window, giving exactly-once delivery.
type Event[T any] interface {
	Subject() string
	MsgID() string
}
