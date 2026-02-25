package models

type AssignOwnerRequest struct {
	AssigneeUserID string `json:"assignee_user_id" binding:"required"`
}

type UnassignOwnerRequest struct {
	ReasonID *int16  `json:"unassign_reason_id"`
	Note     *string `json:"unassign_note"`
}
