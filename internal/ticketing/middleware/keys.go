package middleware

const (
	CtxUserIDKey          = "ticket_user_id"
	CtxActorKey           = "ticket_actor"          // store models.ActorType
	CtxAllowedProjectsKey = "ticket_allowed_projects"
	CtxMaxVisibilityKey   = "ticket_max_visibility" // store models.TicketVisibility
	CtxCapabilitiesKey    = "ticket_capabilities"   // store repo.UserCapabilities
	CtxExternalSubKey     = "ticket_external_subject"
)