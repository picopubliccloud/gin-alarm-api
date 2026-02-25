package models

type ClosureSummaryRequest struct {
	FixHeadline       string
	Symptoms          string
	RootCause         string
	FixApplied        string
	VerificationSteps string
	Prevention        *string
	ResolutionCode    ResolutionCode
}

type CloseTicketRequest struct {
	ResolutionCode    ResolutionCode `json:"resolution_code" binding:"required"`

	FixHeadline       string `json:"fix_headline" binding:"required"`
	Symptoms          string `json:"symptoms" binding:"required"`
	RootCause         string `json:"root_cause" binding:"required"`
	FixApplied        string `json:"fix_applied" binding:"required"`
	VerificationSteps string `json:"verification_steps" binding:"required"`
	Prevention        *string `json:"prevention"`

	// Optional: if ticket is HIGH/CRITICAL, trigger requires a verification attachment
	// Provide metadata so API can create an attachment row before closing.
	VerificationAttachment *VerificationAttachmentInput `json:"verification_attachment"`
}

type VerificationAttachmentInput struct {
	Bucket  string `json:"bucket" binding:"required"`
	Key     string `json:"key" binding:"required"`
	Name    string `json:"file_name" binding:"required"`
	Mime *string `json:"mime_type" binding:"required"`
	Size int64 `json:"size_bytes" binding:"required,gt=0"`
	SHA256Hex *string `json:"sha256_hex"` // optional, store if you want later
}
