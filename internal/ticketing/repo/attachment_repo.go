package repo

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/picopubliccloud/alarm-api/internal/ticketing/models"
)

type AttachmentRepo struct{ db *sql.DB }

func NewAttachmentRepo(db *sql.DB) *AttachmentRepo { return &AttachmentRepo{db: db} }

/* ============================================================
   Existing: Insert verification attachment during CLOSE flow
   (uses tx because close flow is transactional)
============================================================ */

func (r *AttachmentRepo) InsertVerification(
	ctx context.Context,
	tx *sql.Tx,
	ticketID string,
	uploadedBy string,
	in *models.VerificationAttachmentInput,
) error {
	if tx == nil {
		return errors.New("tx is required")
	}

	var shaBytes []byte
	if in.SHA256Hex != nil && *in.SHA256Hex != "" {
		b, err := hex.DecodeString(*in.SHA256Hex)
		if err != nil {
			return err
		}
		shaBytes = b
	}

	const q = `
INSERT INTO ops.attachments (
  ticket_id, uploaded_by, visibility,
  object_store_bucket, object_store_key,
  file_name, mime_type, size_bytes, sha256,
  is_verification, created_at
)
VALUES ($1,$2,'INTERNAL',$3,$4,$5,$6,$7,$8,true,now());
`
	_, err := tx.ExecContext(ctx, q,
		ticketID,
		uploadedBy,
		in.Bucket,
		in.Key,
		in.Name, // matches your model: Name, Mime, Size
		in.Mime,
		in.Size,
		bytesOrNil(shaBytes),
	)
	return err
}

/* ============================================================
   NEW: Generic create/list/get for UI attachment flows
   (register after upload, list attachments, get by id)
============================================================ */

type CreateAttachmentParams struct {
	TicketID       string
	UploadedBy     string
	Visibility     string // "INTERNAL" | "PUBLIC" | "RESTRICTED" (enum in DB)
	Bucket         string
	Key            string
	FileName       string
	MimeType       string
	SizeBytes      int64
	SHA256Hex      *string
	IsVerification bool
}

type AttachmentRow struct {
	AttachmentID   string         `json:"attachment_id"`
	TicketID       string         `json:"ticket_id"`
	UploadedBy     string         `json:"uploaded_by"`
	Visibility     string         `json:"visibility"`
	Bucket         string         `json:"bucket"`
	Key            string         `json:"key"`
	FileName       string         `json:"file_name"`
	MimeType  *string `json:"mime_type,omitempty"`
	SizeBytes *int64  `json:"size_bytes,omitempty"`
	IsVerification bool           `json:"is_verification"`
	CreatedAt      time.Time      `json:"created_at"`
}

// Create inserts into ops.attachments (non-transactional helper).
// Use CreateTx if you need it inside an existing tx.
func (r *AttachmentRepo) Create(ctx context.Context, p CreateAttachmentParams) (*AttachmentRow, error) {
	return r.createWith(ctx, r.db, p)
}

func (r *AttachmentRepo) CreateTx(ctx context.Context, tx *sql.Tx, p CreateAttachmentParams) (*AttachmentRow, error) {
	if tx == nil {
		return nil, errors.New("tx is required")
	}
	return r.createWith(ctx, tx, p)
}

type execer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (r *AttachmentRepo) createWith(ctx context.Context, qx execer, p CreateAttachmentParams) (*AttachmentRow, error) {
	if p.Visibility == "" {
		p.Visibility = "INTERNAL"
	}

	var shaBytes []byte
	if p.SHA256Hex != nil && *p.SHA256Hex != "" {
		b, err := hex.DecodeString(*p.SHA256Hex)
		if err != nil {
			return nil, err
		}
		shaBytes = b
	}

	// Matches your DDL columns exactly:
	// attachment_id uuid default gen_random_uuid()
	const q = `
INSERT INTO ops.attachments (
  ticket_id, uploaded_by, visibility,
  object_store_bucket, object_store_key,
  file_name, mime_type, size_bytes, sha256,
  is_verification, created_at
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,now())
RETURNING
  attachment_id,
  ticket_id,
  uploaded_by,
  visibility,
  object_store_bucket,
  object_store_key,
  file_name,
  mime_type,
  size_bytes,
  is_verification,
  created_at;
`

	row := qx.QueryRowContext(ctx, q,
		p.TicketID,
		p.UploadedBy,
		p.Visibility,
		p.Bucket,
		p.Key,
		p.FileName,
		nullString(p.MimeType),
		nullInt64(p.SizeBytes),
		bytesOrNil(shaBytes),
		p.IsVerification,
	)

	var out AttachmentRow
	var mime sql.NullString
	var size sql.NullInt64

	if err := row.Scan(
	&out.AttachmentID,
	&out.TicketID,
	&out.UploadedBy,
	&out.Visibility,
	&out.Bucket,
	&out.Key,
	&out.FileName,
	&mime,
	&size,
	&out.IsVerification,
	&out.CreatedAt,
	); err != nil {
	return nil, err
	}

	if mime.Valid {
	out.MimeType = &mime.String
	}
	if size.Valid {
	v := size.Int64
	out.SizeBytes = &v
	}

	return &out, nil
}

func (r *AttachmentRepo) ListByTicket(ctx context.Context, ticketID string) ([]AttachmentRow, error) {
	const q = `
SELECT
  attachment_id,
  ticket_id,
  uploaded_by,
  visibility,
  object_store_bucket,
  object_store_key,
  file_name,
  mime_type,
  size_bytes,
  is_verification,
  created_at
FROM ops.attachments
WHERE ticket_id = $1
ORDER BY created_at DESC;
`
	rows, err := r.db.QueryContext(ctx, q, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AttachmentRow
	for rows.Next() {
	var a AttachmentRow
	var mime sql.NullString
	var size sql.NullInt64

	if err := rows.Scan(
		&a.AttachmentID,
		&a.TicketID,
		&a.UploadedBy,
		&a.Visibility,
		&a.Bucket,
		&a.Key,
		&a.FileName,
		&mime,
		&size,
		&a.IsVerification,
		&a.CreatedAt,
	); err != nil {
		return nil, err
	}

	if mime.Valid {
		a.MimeType = &mime.String
	}
	if size.Valid {
		v := size.Int64
		a.SizeBytes = &v
	}

	out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *AttachmentRepo) GetByID(ctx context.Context, ticketID string, attachmentID string) (*AttachmentRow, error) {
	const q = `
SELECT
  attachment_id,
  ticket_id,
  uploaded_by,
  visibility,
  object_store_bucket,
  object_store_key,
  file_name,
  mime_type,
  size_bytes,
  is_verification,
  created_at
FROM ops.attachments
WHERE ticket_id = $1 AND attachment_id = $2
LIMIT 1;
`
	var a AttachmentRow
	var mime sql.NullString
	var size sql.NullInt64

	err := r.db.QueryRowContext(ctx, q, ticketID, attachmentID).Scan(
	&a.AttachmentID,
	&a.TicketID,
	&a.UploadedBy,
	&a.Visibility,
	&a.Bucket,
	&a.Key,
	&a.FileName,
	&mime,
	&size,
	&a.IsVerification,
	&a.CreatedAt,
	)
	if err != nil {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	return nil, err
	}

	if mime.Valid {
	a.MimeType = &mime.String
	}
	if size.Valid {
	v := size.Int64
	a.SizeBytes = &v
	}

	return &a, nil
}

/* ============================================================
   helpers
============================================================ */

func bytesOrNil(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullInt64(v int64) any {
	if v <= 0 {
		return nil
	}
	return v
}
