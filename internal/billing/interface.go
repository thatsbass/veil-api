package billing

// PaymentProvider verifies and parses inbound payment webhook events.
// Implementations live in sub-packages (e.g. internal/billing/stripe).
type PaymentProvider interface {
	// VerifyWebhook authenticates the payload and returns a neutral PaymentEvent.
	VerifyWebhook(payload []byte, sig string) (*PaymentEvent, error)
	// SignatureHeader returns the HTTP header name carrying the webhook signature.
	SignatureHeader() string
}

// PaymentEventType is a provider-agnostic event classification.
type PaymentEventType string

const (
	EventSubscriptionUpdated PaymentEventType = "subscription.updated"
	EventSubscriptionDeleted PaymentEventType = "subscription.deleted"
	EventPaymentSucceeded    PaymentEventType = "payment.succeeded"
	EventUnknown             PaymentEventType = "unknown"
)

// PaymentEvent holds the normalized data extracted from any provider webhook.
type PaymentEvent struct {
	Type               PaymentEventType
	PaymentCustomerID  string // provider customer ID (e.g. cus_xxx)
	PlanID             string // target plan, if present
	InvoiceID          string // invoice ID, if present
}
