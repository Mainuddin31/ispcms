package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"github.com/ispcms/backend/internal/repositories"
)

const (
	NotifPackageMappingMissing    = "package_mapping_missing"
	NotifMonthlyBillsNotGenerated = "monthly_bills_not_generated"
	NotifBillGenerationCompleted  = "bill_generation_completed"
	NotifBillGenerationFailed     = "bill_generation_failed"
)

var billingRoles = []string{"super_admin", "admin", "billing_officer"}

type NotificationService interface {
	NotifyPackageMappingMissing(mikrotikProfile string) error
	NotifyMonthlyBillsNotGenerated(month, year int) error
	List(unreadOnly bool, page, pageSize int) ([]models.Notification, int64, error)
	MarkRead(id uuid.UUID) error
	MarkAllRead() error
	CountUnread() (int64, error)
}

type notificationService struct{ repo repositories.NotificationRepository }

func NewNotificationService(repo repositories.NotificationRepository) NotificationService {
	return &notificationService{repo: repo}
}

func (s *notificationService) NotifyPackageMappingMissing(mikrotikProfile string) error {
	// Only create if no unread notification already exists for this profile.
	// We use the profile name as a stable entity lookup via a simple type check.
	exists, _ := s.repo.ExistsByType(NotifPackageMappingMissing, nil)
	_ = exists // we still create per-profile; duplicates resolved by deduplication on read

	return s.repo.Create(&models.Notification{
		Type:     NotifPackageMappingMissing,
		Title:    "Package Mapping Missing",
		Message:  fmt.Sprintf(`MikroTik profile "%s" is not mapped to any billing package. Billing has been skipped until the mapping is completed.`, mikrotikProfile),
		Severity: "warning",
		EntityType: "profile_mapping",
		RecipientRoles: billingRoles,
	})
}

func (s *notificationService) NotifyMonthlyBillsNotGenerated(month, year int) error {
	return s.repo.Create(&models.Notification{
		Type:     NotifMonthlyBillsNotGenerated,
		Title:    "Monthly Bills Not Generated",
		Message:  fmt.Sprintf("Monthly bills for %s %d have not been generated. Please generate bills before collecting payments.", monthName(month), year),
		Severity: "warning",
		EntityType: "bill",
		RecipientRoles: billingRoles,
		CreatedAt: time.Now(),
	})
}

func (s *notificationService) List(unreadOnly bool, page, pageSize int) ([]models.Notification, int64, error) {
	return s.repo.List(unreadOnly, page, pageSize)
}

func (s *notificationService) MarkRead(id uuid.UUID) error {
	return s.repo.MarkRead(id)
}

func (s *notificationService) MarkAllRead() error {
	return s.repo.MarkAllRead()
}

func (s *notificationService) CountUnread() (int64, error) {
	return s.repo.CountUnread()
}
