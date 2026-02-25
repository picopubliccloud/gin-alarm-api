package service

import (
	"context"

	"github.com/picopubliccloud/alarm-api/internal/ticketing/models"
)

type SearchService struct{}

func NewSearchService() *SearchService { return &SearchService{} }

func (s *SearchService) Search(ctx context.Context, req *models.SearchRequest) ([]models.SearchResult, error) {
	// placeholder until OpenSearch/SQL search implemented
	return []models.SearchResult{}, nil
}
