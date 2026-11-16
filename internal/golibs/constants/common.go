package constants

type ContextKey string

const (
	ContextKeyUserID   ContextKey = "user_id"
	ContextKeyUserRole ContextKey = "user_role"

	FormatISO8601 = "2006-01-02T15:04:05Z07:00"
	FormatDate    = "2006-01-02"
)
