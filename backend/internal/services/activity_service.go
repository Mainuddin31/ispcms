package services

import (
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"github.com/ispcms/backend/internal/repositories"
)

type ActivityService interface {
	Log(userID *uuid.UUID, module, activityType, title, description, refType, refID string)
	List(module, period string, limit int) ([]models.ActivityLog, error)
}

type activityService struct {
	repo repositories.ActivityLogRepository
}

func NewActivityService(repo repositories.ActivityLogRepository) ActivityService {
	return &activityService{repo: repo}
}

func (s *activityService) Log(
	userID *uuid.UUID,
	module, activityType, title, description, refType, refID string,
) {
	a := &models.ActivityLog{
		UserID:        userID,
		Module:        module,
		ActivityType:  activityType,
		Title:         title,
		Description:   description,
		ReferenceType: refType,
		ReferenceID:   refID,
		Action:        activityType,
		Details:       description,
	}
	_ = s.repo.Create(a)
}

func (s *activityService) List(module, period string, limit int) ([]models.ActivityLog, error) {
	return s.repo.List(repositories.ActivityFilter{
		Module: module,
		Period: period,
		Limit:  limit,
	})
}
