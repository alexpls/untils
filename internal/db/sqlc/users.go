package sqlc

func (row *ActiveUserIntegrationsRow) DisplayName() string {
	switch row.Name {
	case NotifierEmail:
		return "Email"
	case NotifierPushover:
		return "Pushover"
	default:
		panic("unhandled name: " + row.Name)
	}
}
