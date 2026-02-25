package handlers

import (
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"

	tMW "github.com/picopubliccloud/alarm-api/internal/ticketing/middleware"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/repo"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/storage"
)

type AttachmentsHandler struct {
	Store *storage.S3Store
	Repo  *repo.AttachmentRepo
}

func NewAttachmentsHandler(store *storage.S3Store, r *repo.AttachmentRepo) *AttachmentsHandler {
	return &AttachmentsHandler{Store: store, Repo: r}
}

type createUploadTempURLReq struct {
	FileName       string `json:"file_name" binding:"required"`
	MimeType       string `json:"mime_type"`
	SizeBytes      int64  `json:"size_bytes" binding:"required"`
	IsVerification bool   `json:"is_verification"`
}

func (h *AttachmentsHandler) CreateUploadTempURL(c *gin.Context) {
	ticketID := c.Param("ticket_id")

	var req createUploadTempURLReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ct := storage.DetectContentType(req.FileName, req.MimeType)
	key := h.Store.MakeTicketKey(ticketID, req.FileName)

	ps, err := h.Store.Presign.PresignPutObject(c.Request.Context(), &s3.PutObjectInput{
	Bucket:      aws.String(h.Store.Bucket),
	Key:         aws.String(key),
	ContentType: aws.String(ct),
	}, func(opt *s3.PresignOptions) {
	opt.Expires = 15 * time.Minute
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "presign upload failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upload_url": ps.URL,
		"bucket":     h.Store.Bucket,
		"key":        key,
		"mime_type":  ct,
		"expires_in": 900,
	})
}

type registerAttachmentReq struct {
	Bucket         string `json:"bucket" binding:"required"`
	Key            string `json:"key" binding:"required"`
	FileName       string `json:"file_name" binding:"required"`
	MimeType       string `json:"mime_type"`
	SizeBytes      int64  `json:"size_bytes" binding:"required"`
	IsVerification bool   `json:"is_verification"`
	Visibility     string `json:"visibility"` // optional; default INTERNAL
}

func (h *AttachmentsHandler) RegisterAttachment(c *gin.Context) {
	ticketID := c.Param("ticket_id")

	var req registerAttachmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString(tMW.CtxUserIDKey)

	vis := req.Visibility
	if vis == "" {
		vis = "INTERNAL"
	}

	att, err := h.Repo.Create(c.Request.Context(), repo.CreateAttachmentParams{
		TicketID:        ticketID,
		UploadedBy:      userID, 
		Visibility:      vis,
		Bucket:          req.Bucket,
		Key:             req.Key,
		FileName:        req.FileName,
		MimeType:        req.MimeType,
		SizeBytes:       req.SizeBytes,
		IsVerification:  req.IsVerification,
		SHA256Hex:       nil, // optional (add later if you compute it client-side)
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "register failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"attachment": att})
}

func (h *AttachmentsHandler) ListAttachments(c *gin.Context) {
	ticketID := c.Param("ticket_id")

	items, err := h.Repo.ListByTicket(c.Request.Context(), ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed", "details": err.Error()})
		return
	}

	// Your frontend expects either array OR {items:[]}, you're already handling both.
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *AttachmentsHandler) GetDownloadTempURL(c *gin.Context) {
	ticketID := c.Param("ticket_id")
	attachmentID := c.Param("attachment_id")

	att, err := h.Repo.GetByID(c.Request.Context(), ticketID, attachmentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found", "details": err.Error()})
		return
	}

	ps, err := h.Store.Presign.PresignGetObject(c.Request.Context(), &s3.GetObjectInput{
		Bucket: aws.String(att.Bucket),
		Key:    aws.String(att.Key),
	}, func(opt *s3.PresignOptions) {
		opt.Expires = 10 * time.Minute
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "presign download failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": ps.URL})
}
