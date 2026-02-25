package models

type ActorType string

const (
	ActorTypeCustomer ActorType = "CUSTOMER"
	ActorTypeOPS      ActorType = "OPS"
	ActorTypeSystem   ActorType = "SYSTEM"
)

type TicketVisibility string

const (
	VisibilityPublic     TicketVisibility = "PUBLIC"
	VisibilityInternal   TicketVisibility = "INTERNAL"
	VisibilityRestricted TicketVisibility = "RESTRICTED"
)

type TicketStatus string

const (
	StatusNew            TicketStatus = "NEW"
	StatusAcknowledged   TicketStatus = "ACKNOWLEDGED"
	StatusInProgress     TicketStatus = "IN_PROGRESS"
	StatusWaitingCust    TicketStatus = "WAITING_CUSTOMER"
	StatusWaitingVendor  TicketStatus = "WAITING_VENDOR"
	StatusMitigated      TicketStatus = "MITIGATED"
	StatusResolved       TicketStatus = "RESOLVED"
	StatusClosed         TicketStatus = "CLOSED"
)

type TicketSeverity string

const (
	SevLow      TicketSeverity = "LOW"
	SevMedium   TicketSeverity = "MEDIUM"
	SevHigh     TicketSeverity = "HIGH"
	SevCritical TicketSeverity = "CRITICAL"
)

type TicketType string

const (
	TypeIncident        TicketType = "INCIDENT"
	TypeServiceRequest  TicketType = "SERVICE_REQUEST"
	TypeQuestion        TicketType = "QUESTION"
	TypeSecurityIncident TicketType = "SECURITY_INCIDENT"
	TypeChangeRequest   TicketType = "CHANGE_REQUEST"
)

type AssignmentRole string

const (
	RoleOwner        AssignmentRole = "OWNER"
	RoleWatcher      AssignmentRole = "WATCHER"
	RoleCollaborator AssignmentRole = "COLLABORATOR"
)

type UpdateType string

const (
	UpdateComment      UpdateType = "COMMENT"
	UpdateStatusChange UpdateType = "STATUS_CHANGE"
	UpdateFieldChange  UpdateType = "FIELD_CHANGE"
	UpdateSystemEvent  UpdateType = "SYSTEM_EVENT"
)

type ResolutionCode string

const (
	ResFixed                 ResolutionCode = "FIXED"
	ResWorkaround            ResolutionCode = "WORKAROUND"
	ResCustomerActionReq     ResolutionCode = "CUSTOMER_ACTION_REQUIRED"
	ResDuplicate             ResolutionCode = "DUPLICATE"
	ResCannotReproduce       ResolutionCode = "CANNOT_REPRODUCE"
	ResExternalDependency    ResolutionCode = "EXTERNAL_DEPENDENCY"
	ResNotABug               ResolutionCode = "NOT_A_BUG"
	ResWontFix               ResolutionCode = "WONT_FIX"
)
