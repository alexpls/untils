package models

func (row *UserIntegrationsRow) DisplayName() string {
	switch row.Name {
	case NotifierEmail:
		return "Email"
	case NotifierPushover:
		return "Pushover"
	default:
		panic("unhandled name: " + row.Name)
	}
}
