package handler

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	paymentapp "marketplace-backend/internal/app/payment"
	"marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
)

const maxDocumentSize = 10 << 20 // 10MB

type IdentityDocumentHandler struct {
	paymentSvc *paymentapp.Service
}

func NewIdentityDocumentHandler(paymentSvc *paymentapp.Service) *IdentityDocumentHandler {
	return &IdentityDocumentHandler{paymentSvc: paymentSvc}
}

func (h *IdentityDocumentHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxDocumentSize)
	if err := r.ParseMultipartForm(maxDocumentSize); err != nil {
		res.Error(w, http.StatusBadRequest, "file_too_large", "file must be under 10MB")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		res.Error(w, http.StatusBadRequest, "missing_file", "file is required")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if !isValidDocumentType(contentType) {
		res.Error(w, http.StatusBadRequest, "invalid_file_type", "accepted: JPG, PNG, PDF")
		return
	}

	category := r.FormValue("category")
	docType := r.FormValue("document_type")
	side := r.FormValue("side")

	// Buffer the file for dual upload (R2 + Stripe)
	data, err := io.ReadAll(file)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "read_error", "cannot read file")
		return
	}

	doc, err := h.paymentSvc.UploadIdentityDocument(r.Context(), userID, paymentapp.UploadIdentityDocumentInput{
		Category:     category,
		DocumentType: docType,
		Side:         side,
		Filename:     header.Filename,
		ContentType:  contentType,
		FileData:     data,
	})
	if err != nil {
		handleIdentityDocError(w, err)
		return
	}

	fileURL := h.paymentSvc.GetDocumentFileURL(doc.FileKey)
	res.JSON(w, http.StatusCreated, response.NewIdentityDocumentResponse(doc, fileURL))
}

func (h *IdentityDocumentHandler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	docs, err := h.paymentSvc.ListIdentityDocuments(r.Context(), userID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	result := make([]response.IdentityDocumentResponse, len(docs))
	for i, doc := range docs {
		result[i] = response.NewIdentityDocumentResponse(doc, h.paymentSvc.GetDocumentFileURL(doc.FileKey))
	}

	res.JSON(w, http.StatusOK, result)
}

func (h *IdentityDocumentHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	docID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.paymentSvc.DeleteIdentityDocument(r.Context(), userID, docID); err != nil {
		handleIdentityDocError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func isValidDocumentType(ct string) bool {
	return strings.HasPrefix(ct, "image/") || ct == "application/pdf"
}

func handleIdentityDocError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, payment.ErrInvalidDocumentCategory),
		errors.Is(err, payment.ErrInvalidDocumentType),
		errors.Is(err, payment.ErrInvalidDocumentSide),
		errors.Is(err, payment.ErrDocumentFileKeyRequired):
		res.Error(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, payment.ErrDocumentNotFound):
		res.Error(w, http.StatusNotFound, "document_not_found", err.Error())
	case errors.Is(err, payment.ErrStripeAccountNotFound):
		res.Error(w, http.StatusBadRequest, "no_stripe_account", "complete payment info first")
	default:
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
