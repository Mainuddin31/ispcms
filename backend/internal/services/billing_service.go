package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"github.com/ispcms/backend/internal/repositories"
	"gorm.io/gorm"
)

// ── Package Service ──────────────────────────────────────────────────────────

type PackageService interface {
	Create(p *models.Package) error
	Update(id uuid.UUID, body map[string]interface{}) (*models.Package, error)
	Delete(id uuid.UUID) error
	Get(id uuid.UUID) (*models.Package, error)
	List(filter repositories.PackageFilter, page, pageSize int) ([]models.Package, int64, error)
	ListActive() ([]models.Package, error)
}

type packageService struct{ repo repositories.PackageRepository }

func NewPackageService(repo repositories.PackageRepository) PackageService {
	return &packageService{repo: repo}
}

func (s *packageService) Create(p *models.Package) error {
	if _, err := s.repo.FindByName(p.PackageName); err == nil {
		return errors.New("package name already exists")
	}
	return s.repo.Create(p)
}

func (s *packageService) Update(id uuid.UUID, body map[string]interface{}) (*models.Package, error) {
	p, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if name, ok := body["package_name"].(string); ok && name != p.PackageName {
		if _, err := s.repo.FindByName(name); err == nil {
			return nil, errors.New("package name already exists")
		}
		p.PackageName = name
	}
	if v, ok := body["display_name"].(string); ok {
		p.DisplayName = v
	}
	if v, ok := body["speed"].(string); ok {
		p.Speed = v
	}
	if v, ok := body["monthly_price"].(float64); ok {
		p.MonthlyPrice = v
	}
	if v, ok := body["vat_percent"].(float64); ok {
		p.VATPercent = v
	}
	if v, ok := body["installation_charge"].(float64); ok {
		p.InstallationCharge = v
	}
	if v, ok := body["description"].(string); ok {
		p.Description = v
	}
	if v, ok := body["status"].(string); ok {
		p.Status = v
	}
	return p, s.repo.Update(p)
}

func (s *packageService) Delete(id uuid.UUID) error { return s.repo.Delete(id) }
func (s *packageService) Get(id uuid.UUID) (*models.Package, error) { return s.repo.FindByID(id) }
func (s *packageService) List(filter repositories.PackageFilter, page, pageSize int) ([]models.Package, int64, error) {
	return s.repo.List(filter, page, pageSize)
}
func (s *packageService) ListActive() ([]models.Package, error) { return s.repo.ListActive() }

// ── Profile Mapping Service ──────────────────────────────────────────────────

type ProfileMappingService interface {
	Create(m *models.ProfileMapping) error
	Update(id uuid.UUID, body map[string]interface{}) (*models.ProfileMapping, error)
	Delete(id uuid.UUID) error
	Get(id uuid.UUID) (*models.ProfileMapping, error)
	List(filter repositories.ProfileMappingFilter, page, pageSize int) ([]models.ProfileMapping, int64, error)
	UnmappedProfiles() ([]string, error)
}

type profileMappingService struct{ repo repositories.ProfileMappingRepository }

func NewProfileMappingService(repo repositories.ProfileMappingRepository) ProfileMappingService {
	return &profileMappingService{repo: repo}
}

func (s *profileMappingService) Create(m *models.ProfileMapping) error {
	if _, err := s.repo.FindByProfile(m.MikrotikProfile); err == nil {
		return errors.New("profile already mapped")
	}
	return s.repo.Create(m)
}

func (s *profileMappingService) Update(id uuid.UUID, body map[string]interface{}) (*models.ProfileMapping, error) {
	m, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if v, ok := body["package_id"].(string); ok {
		uid, err := uuid.Parse(v)
		if err != nil {
			return nil, errors.New("invalid package_id")
		}
		m.PackageID = uid
	}
	if v, ok := body["notes"].(string); ok {
		m.Notes = v
	}
	return m, s.repo.Update(m)
}

func (s *profileMappingService) Delete(id uuid.UUID) error { return s.repo.Delete(id) }
func (s *profileMappingService) Get(id uuid.UUID) (*models.ProfileMapping, error) {
	return s.repo.FindByID(id)
}
func (s *profileMappingService) List(filter repositories.ProfileMappingFilter, page, pageSize int) ([]models.ProfileMapping, int64, error) {
	return s.repo.List(filter, page, pageSize)
}
func (s *profileMappingService) UnmappedProfiles() ([]string, error) {
	return s.repo.UnmappedProfiles()
}

// ── Billing Service ──────────────────────────────────────────────────────────

type GenerateBillsRequest struct {
	Month       int
	Year        int
	GeneratedBy *uuid.UUID
}

type AutoAssignResult struct {
	Assigned int `json:"assigned"`
	Skipped  int `json:"skipped"` // already had subscription or no mapping
	Total    int `json:"total"`
}

type BillingService interface {
	GenerateBills(req GenerateBillsRequest) (*models.BillGenerationLog, error)
	GetBill(id uuid.UUID) (*models.MonthlyBill, error)
	ListBills(filter repositories.BillFilter, page, pageSize int) ([]models.MonthlyBill, int64, error)
	UpdateBillStatus(id uuid.UUID, status string, paidAmount *float64, notes, paymentMethod, receiptNumber string, receivedByID *uuid.UUID) (*models.MonthlyBill, error)
	GetPaymentHistory(internetAccountID uuid.UUID) ([]models.PaymentRecord, error)
	GetBillingHistory(internetAccountID uuid.UUID) ([]BillingHistoryEntry, error)
	// AssignSubscription manually assigns a package to an account.
	AssignSubscription(internetAccountID, packageID uuid.UUID) (*models.CustomerSubscription, error)
	// AssignSubscriptionFromSync is called during sync — same logic but accepts package directly.
	AssignSubscriptionFromSync(internetAccountID uuid.UUID, pkg *models.Package) (*models.CustomerSubscription, error)
	// AutoAssignFromProfiles assigns subscriptions to all active accounts that have a profile
	// mapping but no active subscription. Safe to call at any time.
	AutoAssignFromProfiles() (*AutoAssignResult, error)
	GetActiveSubscription(internetAccountID uuid.UUID) (*models.CustomerSubscription, error)
	ListSubscriptions(page, pageSize int, accountID *uuid.UUID, packageID *uuid.UUID, activeOnly bool) ([]models.CustomerSubscription, int64, error)
	CheckMonthlyBillingStatus(month, year int) (generated, pending int64, err error)
	ListGenerationLogs(month, year, limit int) ([]models.BillGenerationLog, error)
}

type billingService struct {
	packageRepo repositories.PackageRepository
	subRepo     repositories.SubscriptionRepository
	billRepo    repositories.BillRepository
	notifRepo   repositories.NotificationRepository
	paymentRepo repositories.PaymentRepository
	activitySvc ActivityService
	db          *gorm.DB
}

func NewBillingService(
	packageRepo repositories.PackageRepository,
	subRepo repositories.SubscriptionRepository,
	billRepo repositories.BillRepository,
	notifRepo repositories.NotificationRepository,
	paymentRepo repositories.PaymentRepository,
	activitySvc ActivityService,
	db *gorm.DB,
) BillingService {
	return &billingService{
		packageRepo: packageRepo,
		subRepo:     subRepo,
		billRepo:    billRepo,
		notifRepo:   notifRepo,
		paymentRepo: paymentRepo,
		activitySvc: activitySvc,
		db:          db,
	}
}

// GenerateBills creates one bill per active-subscription customer for the current month only.
// Rules:
//   - Only the current calendar month is allowed — past/future months are rejected.
//   - Uses today as the subscription lookup date so accounts subscribed any time
//     this month are found (not just those subscribed before the 1st).
//   - Skips duplicate bills (unique DB constraint + pre-check).
//   - Skips accounts without a subscription.
//   - Logs every skip with a reason.
func (s *billingService) GenerateBills(req GenerateBillsRequest) (*models.BillGenerationLog, error) {
	now := time.Now().UTC()
	// Enforce: only the current running month can be billed.
	if req.Month != int(now.Month()) || req.Year != now.Year() {
		return nil, fmt.Errorf(
			"bills can only be generated for the current month (%s %d)",
			monthName(int(now.Month())), now.Year(),
		)
	}

	// Use today as the lookup date — subscriptions assigned any time this month are found.
	billingDate := now
	dueDate := time.Date(req.Year, time.Month(req.Month+1), 0, 23, 59, 59, 0, time.UTC) // last day of month

	var accounts []models.InternetAccount
	s.db.Where("archived_at IS NULL AND disabled = false").Find(&accounts)

	genLog := &models.BillGenerationLog{
		BillingMonth:  req.Month,
		BillingYear:   req.Year,
		TotalAccounts: len(accounts),
		GeneratedByID: req.GeneratedBy,
		GeneratedAt:   time.Now(),
	}
	var skipDetails []models.BillSkipDetail

	skip := func(acc models.InternetAccount, reason string) {
		skipDetails = append(skipDetails, models.BillSkipDetail{
			AccountID: acc.ID.String(),
			Username:  acc.Username,
			Reason:    reason,
		})
	}

	for _, acc := range accounts {
		// Find subscription active on the 1st of the billing month.
		sub, err := s.subRepo.FindOnDate(acc.ID, billingDate)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				skip(acc, "No subscription active on billing date")
			} else {
				skip(acc, fmt.Sprintf("Subscription lookup error: %v", err))
			}
			continue
		}

		// Duplicate check.
		exists, _ := s.billRepo.ExistsByAccountMonth(acc.ID, req.Month, req.Year)
		if exists {
			skip(acc, "Bill already exists for this month")
			continue
		}

		monthlyCharge := sub.MonthlyPrice
		vatAmount := monthlyCharge * (sub.Package.VATPercent / 100.0)
		totalAmount := monthlyCharge + vatAmount

		billNum, _ := s.billRepo.GenerateNextBillNumber(req.Month, req.Year)
		bill := &models.MonthlyBill{
			BillNumber:        billNum,
			InternetAccountID: acc.ID,
			PackageID:         sub.PackageID,
			SubscriptionID:    sub.ID,
			BillingMonth:      req.Month,
			BillingYear:       req.Year,
			MonthlyCharge:     monthlyCharge,
			VAT:               vatAmount,
			TotalAmount:       totalAmount,
			DueAmount:         totalAmount,
			Status:            "pending",
			DueDate:           &dueDate,
			GeneratedAt:       time.Now(),
		}

		if err := s.billRepo.Create(bill); err != nil {
			skip(acc, fmt.Sprintf("Failed to create bill: %v", err))
			continue
		}
		genLog.BillsGenerated++
	}

	genLog.BillsSkipped = len(skipDetails)
	genLog.SkipDetails = skipDetails
	genLog.Status = "completed"
	if genLog.BillsGenerated == 0 && genLog.TotalAccounts > 0 {
		genLog.Status = "partial"
	}

	_ = s.billRepo.CreateGenerationLog(genLog)
	if s.activitySvc != nil {
		desc := fmt.Sprintf("%s %d: %d bills generated, %d skipped", monthName(req.Month), req.Year, genLog.BillsGenerated, genLog.BillsSkipped)
		s.activitySvc.Log(req.GeneratedBy, "billing", "bills_generated", "Bills Generated", desc, "bill_generation_log", genLog.ID.String())
	}

	// Notify on completion.
	_ = s.notifRepo.Create(&models.Notification{
		Type:           "bill_generation_completed",
		Title:          "Bill Generation Completed",
		Message:        fmt.Sprintf("Monthly bills for %s %d: %d generated, %d skipped.", monthName(req.Month), req.Year, genLog.BillsGenerated, genLog.BillsSkipped),
		Severity:       "info",
		EntityType:     "bill",
		RecipientRoles: []string{"super_admin", "admin", "billing_officer"},
	})

	return genLog, nil
}

func (s *billingService) GetBill(id uuid.UUID) (*models.MonthlyBill, error) {
	return s.billRepo.FindByID(id)
}

func (s *billingService) ListBills(filter repositories.BillFilter, page, pageSize int) ([]models.MonthlyBill, int64, error) {
	return s.billRepo.List(filter, page, pageSize)
}

func (s *billingService) UpdateBillStatus(id uuid.UUID, status string, paidAmount *float64, notes, paymentMethod, receiptNumber string, receivedByID *uuid.UUID) (*models.MonthlyBill, error) {
	bill, err := s.billRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if status != "" {
		bill.Status = status
	}
	if notes != "" {
		bill.Notes = notes
	}
	var paymentAmount float64
	if paidAmount != nil {
		// paidAmount is the new running total (cumulative paid so far).
		paymentAmount = *paidAmount - bill.PaidAmount // amount in this transaction
		bill.PaidAmount = *paidAmount
		bill.DueAmount = bill.TotalAmount - bill.PaidAmount
		if bill.PaidAmount <= 0 {
			bill.Status = "pending"
		} else if bill.PaidAmount >= bill.TotalAmount {
			bill.Status = "paid"
			bill.DueAmount = 0
		} else {
			bill.Status = "partial"
		}
	}
	if err := s.billRepo.Update(bill); err != nil {
		return nil, err
	}
	// Record a PaymentRecord whenever actual money is being received (positive delta).
	if paymentAmount > 0 {
		method := paymentMethod
		if method == "" {
			method = "cash"
		}
		rec := &models.PaymentRecord{
			BillID:            bill.ID,
			InternetAccountID: bill.InternetAccountID,
			Amount:            paymentAmount,
			Notes:             notes,
			PaymentMethod:     method,
			ReceiptNumber:     receiptNumber,
			ReceivedByID:      receivedByID,
		}
		_ = s.paymentRepo.Create(rec)
		if s.activitySvc != nil {
			desc := fmt.Sprintf("Bill: %s | Amount: %.2f | Method: %s", bill.BillNumber, paymentAmount, method)
			s.activitySvc.Log(receivedByID, "billing", "payment_received", "Payment Received", desc, "payment", rec.ID.String())
		}
	}
	return bill, nil
}

func (s *billingService) GetPaymentHistory(internetAccountID uuid.UUID) ([]models.PaymentRecord, error) {
	return s.paymentRepo.ListByAccount(internetAccountID)
}

// BillingHistoryEntry combines a monthly bill with its latest payment record info.
type BillingHistoryEntry struct {
	models.MonthlyBill
	LastPaymentMethod string     `json:"last_payment_method"`
	LastReceiptNumber string     `json:"last_receipt_number"`
	LastCollectedBy   *string    `json:"last_collected_by"`
	LastPaidAt        *time.Time `json:"last_paid_at"`
}

func (s *billingService) GetBillingHistory(internetAccountID uuid.UUID) ([]BillingHistoryEntry, error) {
	bills, _, err := s.billRepo.List(repositories.BillFilter{
		InternetAccountID: &internetAccountID,
	}, 1, 200)
	if err != nil {
		return nil, err
	}

	entries := make([]BillingHistoryEntry, 0, len(bills))
	for _, b := range bills {
		entry := BillingHistoryEntry{MonthlyBill: b}
		// Get latest payment for this bill
		records, _ := s.paymentRepo.ListByBill(b.ID)
		if len(records) > 0 {
			latest := records[0] // already ordered DESC
			entry.LastPaymentMethod = latest.PaymentMethod
			entry.LastReceiptNumber = latest.ReceiptNumber
			entry.LastPaidAt = &latest.PaidAt
			if latest.ReceivedBy != nil {
				name := latest.ReceivedBy.FullName
				entry.LastCollectedBy = &name
			}
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *billingService) AssignSubscription(internetAccountID, packageID uuid.UUID) (*models.CustomerSubscription, error) {
	pkg, err := s.packageRepo.FindByID(packageID)
	if err != nil {
		return nil, fmt.Errorf("package not found")
	}
	return s.AssignSubscriptionFromSync(internetAccountID, pkg)
}

func (s *billingService) AssignSubscriptionFromSync(internetAccountID uuid.UUID, pkg *models.Package) (*models.CustomerSubscription, error) {
	if pkg.Status != "active" {
		return nil, fmt.Errorf("package %s is not active", pkg.PackageName)
	}
	now := time.Now()

	// Check if same package is already active — avoid unnecessary churn.
	current, err := s.subRepo.FindActiveByAccount(internetAccountID)
	if err == nil && current.PackageID == pkg.ID {
		return current, nil // no change
	}

	// Deactivate current (if any).
	_ = s.subRepo.DeactivateForAccount(internetAccountID, now)

	sub := &models.CustomerSubscription{
		InternetAccountID: internetAccountID,
		PackageID:         pkg.ID,
		MonthlyPrice:      pkg.MonthlyPrice,
		EffectiveFrom:     now,
		IsActive:          true,
	}
	err = s.subRepo.Create(sub)
	if err != nil {
		return nil, err
	}
	// Load package relation.
	sub.Package = *pkg
	return sub, nil
}

// AutoAssignFromProfiles loops every active, non-archived internet account.
// If the account has a mapped PPPoE profile but no active subscription, it assigns one.
// Accounts already subscribed (even to a different package) are skipped — use the manual
// assign or sync to change an existing subscription.
func (s *billingService) AutoAssignFromProfiles() (*AutoAssignResult, error) {
	// Load all active profile→package mappings into a map for O(1) lookup.
	var mappings []models.ProfileMapping
	if err := s.db.Preload("Package").Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("loading profile mappings: %w", err)
	}
	mappingByProfile := make(map[string]*models.ProfileMapping, len(mappings))
	for i := range mappings {
		if mappings[i].Package.Status == "active" {
			mappingByProfile[mappings[i].MikrotikProfile] = &mappings[i]
		}
	}

	// Load all active, non-archived accounts.
	var accounts []models.InternetAccount
	if err := s.db.Where("archived_at IS NULL AND disabled = false").Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("loading accounts: %w", err)
	}

	result := &AutoAssignResult{Total: len(accounts)}
	for _, acc := range accounts {
		if acc.Profile == "" {
			result.Skipped++
			continue
		}
		mapping, ok := mappingByProfile[acc.Profile]
		if !ok {
			result.Skipped++
			continue
		}
		// Skip if already has any active subscription (don't override existing billing).
		if _, err := s.subRepo.FindActiveByAccount(acc.ID); err == nil {
			result.Skipped++
			continue
		}
		// No active subscription — assign now.
		if _, err := s.AssignSubscriptionFromSync(acc.ID, &mapping.Package); err == nil {
			result.Assigned++
		} else {
			result.Skipped++
		}
	}
	return result, nil
}

func (s *billingService) GetActiveSubscription(internetAccountID uuid.UUID) (*models.CustomerSubscription, error) {
	return s.subRepo.FindActiveByAccount(internetAccountID)
}

func (s *billingService) ListSubscriptions(page, pageSize int, accountID *uuid.UUID, packageID *uuid.UUID, activeOnly bool) ([]models.CustomerSubscription, int64, error) {
	return s.subRepo.List(page, pageSize, accountID, packageID, activeOnly)
}

func (s *billingService) CheckMonthlyBillingStatus(month, year int) (generated, pending int64, err error) {
	return s.billRepo.SummarizeBilling(month, year)
}

func (s *billingService) ListGenerationLogs(month, year, limit int) ([]models.BillGenerationLog, error) {
	return s.billRepo.ListGenerationLogs(month, year, limit)
}

// monthName returns the full English month name for a month number 1–12.
func monthName(m int) string {
	months := []string{"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}
	if m < 1 || m > 12 {
		return "Unknown"
	}
	return months[m]
}
