package notifications

import "github.com/alexpls/untils/internal/models"

type Capabilities struct {
	EmailEnabled    bool
	PushoverEnabled bool
	WebhookEnabled  bool
}

func (c Capabilities) Enabled(notifier models.Notifier) bool {
	switch notifier {
	case models.NotifierEmail:
		return c.EmailEnabled
	case models.NotifierPushover:
		return c.PushoverEnabled
	case models.NotifierWebhook:
		return c.WebhookEnabled
	default:
		panic("unsupported notifier: " + notifier)
	}
}
