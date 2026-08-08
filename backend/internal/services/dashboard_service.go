package services

import (
	"fmt"
	"time"

	"github.com/ispcms/backend/internal/models"
	"github.com/ispcms/backend/internal/repositories"
	"gorm.io/gorm"
)

// MonthlyPoint is one month of collection + expense data for charts.
type MonthlyPoint struct {
	Year       int     `json:"year"`
	Month      int     `json:"month"`
	Label      string  `json:"label"` // e.g. "Jul 2026"
	Collection float64 `json:"collection"`
	Expense    float64 `json:"expense"`
	CashInHand float64 `json:"cash_in_hand"`
}

type DashboardStats struct {
	TotalRouters    int64      `json:"total_routers"`
	OnlineRouters   int64      `json:"online_routers"`
	OfflineRouters  int64      `json:"offline_routers"`
	TotalSecrets    int64      `json:"total_pppoe_secrets"`
	ActiveSecrets   int64      `json:"active_pppoe_users"`
	DisabledSecrets int64      `json:"disabled_pppoe_users"`
	ActiveSessions  int64      `json:"active_sessions"`
	LastSyncTime    *time.Time `json:"last_sync_time"`

	// Internet account stats
	TotalAccounts    int64 `json:"total_accounts"`
	EnabledAccounts  int64 `json:"enabled_accounts"`
	DisabledAccounts int64 `json:"disabled_accounts"`
	OnlineAccounts   int64 `json:"online_accounts"`
	OfflineAccounts  int64 `json:"offline_accounts"`
	ArchivedAccounts int64 `json:"archived_accounts"`

	// Billing stats
	TotalPackages        int64 `json:"total_packages"`
	ActivePackages       int64 `json:"active_packages"`
	ActiveSubscriptions  int64 `json:"active_subscriptions"`
	UnmappedProfiles     int64 `json:"unmapped_profiles"`
	BillsThisMonth       int64 `json:"bills_this_month"`
	BillsPendingGenerate int64 `json:"bills_pending_generate"`

	// Financial — collections
	TodayCollection     float64 `json:"today_collection"`
	MonthlyCollection   float64 `json:"monthly_collection"`
	LastMonthCollection float64 `json:"last_month_collection"`
	TotalCollection     float64 `json:"total_collection"`
	TotalOutstandingDue float64 `json:"total_outstanding_due"`
	TotalBillsGenerated int64   `json:"total_bills_generated"`
	BillsPaid           int64   `json:"bills_paid"`
	BillsPending        int64   `json:"bills_billing_pending"`

	// Financial — expenses
	TodayExpense   float64 `json:"today_expense"`
	MonthlyExpense float64 `json:"monthly_expense"`
	TotalExpense   float64 `json:"total_expense"`

	// Derived
	CashInHand float64 `json:"cash_in_hand"` // = TotalCollection - TotalExpense (never stored)

	// Charts
	MonthlyChart       []MonthlyPoint   `json:"monthly_chart"`
	ExpenseCategoryPie []models.CategoryTotal `json:"expense_category_pie"`

	RecentSyncLogs   []models.SyncLog     `json:"recent_sync_logs"`
	RecentActivities []models.ActivityLog `json:"recent_activities"`
}

type DashboardService interface {
	GetStats() (*DashboardStats, error)
}

type dashboardService struct {
	routerRepo          repositories.RouterRepository
	pppoeRepo           repositories.PPPoERepository
	internetAccountRepo repositories.InternetAccountRepository
	packageRepo         repositories.PackageRepository
	profileMappingRepo  repositories.ProfileMappingRepository
	subRepo             repositories.SubscriptionRepository
	billRepo            repositories.BillRepository
	expenseRepo         repositories.ExpenseRepository
	activityRepo        repositories.ActivityLogRepository
	db                  *gorm.DB
}

func NewDashboardService(
	routerRepo repositories.RouterRepository,
	pppoeRepo repositories.PPPoERepository,
	internetAccountRepo repositories.InternetAccountRepository,
	packageRepo repositories.PackageRepository,
	profileMappingRepo repositories.ProfileMappingRepository,
	subRepo repositories.SubscriptionRepository,
	billRepo repositories.BillRepository,
	expenseRepo repositories.ExpenseRepository,
	activityRepo repositories.ActivityLogRepository,
	db *gorm.DB,
) DashboardService {
	return &dashboardService{
		routerRepo:          routerRepo,
		pppoeRepo:           pppoeRepo,
		internetAccountRepo: internetAccountRepo,
		packageRepo:         packageRepo,
		profileMappingRepo:  profileMappingRepo,
		subRepo:             subRepo,
		billRepo:            billRepo,
		expenseRepo:         expenseRepo,
		activityRepo:        activityRepo,
		db:                  db,
	}
}

var monthNames = []string{"", "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

func (s *dashboardService) GetStats() (*DashboardStats, error) {
	stats := &DashboardStats{}

	total, online, offline, err := s.routerRepo.CountByStatus()
	if err != nil {
		return nil, err
	}
	stats.TotalRouters = total
	stats.OnlineRouters = online
	stats.OfflineRouters = offline

	secTotal, secActive, secDisabled, err := s.pppoeRepo.CountSecrets()
	if err != nil {
		return nil, err
	}
	stats.TotalSecrets = secTotal
	stats.ActiveSecrets = secActive
	stats.DisabledSecrets = secDisabled

	sessions, err := s.pppoeRepo.CountSessions()
	if err != nil {
		return nil, err
	}
	stats.ActiveSessions = sessions

	iaTotal, iaEnabled, iaDisabled, iaOnline, iaOffline, iaArchived, err :=
		s.internetAccountRepo.CountStats(nil, false)
	if err == nil {
		stats.TotalAccounts = iaTotal
		stats.EnabledAccounts = iaEnabled
		stats.DisabledAccounts = iaDisabled
		stats.OnlineAccounts = iaOnline
		stats.OfflineAccounts = iaOffline
		stats.ArchivedAccounts = iaArchived
	}

	var lastSync models.SyncLog
	if err := s.db.Order("completed_at DESC").First(&lastSync).Error; err == nil {
		stats.LastSyncTime = lastSync.CompletedAt
	}

	var syncLogs []models.SyncLog
	s.db.Preload("Router").Order("started_at DESC").Limit(5).Find(&syncLogs)
	stats.RecentSyncLogs = syncLogs

	// Billing
	if s.packageRepo != nil {
		total, active, _ := s.packageRepo.Count()
		stats.TotalPackages = total
		stats.ActivePackages = active
	}
	if s.subRepo != nil {
		count, _ := s.subRepo.CountActive()
		stats.ActiveSubscriptions = count
	}
	if s.profileMappingRepo != nil {
		profiles, _ := s.profileMappingRepo.UnmappedProfiles()
		stats.UnmappedProfiles = int64(len(profiles))
	}
	if s.billRepo != nil {
		now := time.Now()
		generated, pending, _ := s.billRepo.SummarizeBilling(int(now.Month()), now.Year())
		stats.BillsThisMonth = generated
		stats.BillsPendingGenerate = pending
	}

	// ── Financial dates ───────────────────────────────────────────────────────
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastMonthStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
	lastMonthEnd := monthStart.Add(-time.Nanosecond)

	// ── Collection stats ──────────────────────────────────────────────────────
	s.db.Raw(`SELECT COALESCE(SUM(amount), 0) FROM payment_records WHERE paid_at >= ?`, todayStart).Scan(&stats.TodayCollection)
	s.db.Raw(`SELECT COALESCE(SUM(amount), 0) FROM payment_records WHERE paid_at >= ?`, monthStart).Scan(&stats.MonthlyCollection)
	s.db.Raw(`SELECT COALESCE(SUM(amount), 0) FROM payment_records WHERE paid_at >= ? AND paid_at <= ?`, lastMonthStart, lastMonthEnd).Scan(&stats.LastMonthCollection)
	s.db.Raw(`SELECT COALESCE(SUM(amount), 0) FROM payment_records`).Scan(&stats.TotalCollection)
	s.db.Raw(`SELECT COALESCE(SUM(due_amount), 0) FROM monthly_bills WHERE status NOT IN ('paid', 'cancelled')`).Scan(&stats.TotalOutstandingDue)
	s.db.Model(&models.MonthlyBill{}).Where("status != 'cancelled'").Count(&stats.TotalBillsGenerated)
	s.db.Model(&models.MonthlyBill{}).Where("status = 'paid'").Count(&stats.BillsPaid)
	s.db.Model(&models.MonthlyBill{}).Where("status IN ('pending', 'due', 'partial')").Count(&stats.BillsPending)

	// ── Expense stats ─────────────────────────────────────────────────────────
	if s.expenseRepo != nil {
		todayEnd := todayStart.Add(24 * time.Hour)
		stats.TodayExpense, _ = s.expenseRepo.Summary(&todayStart, &todayEnd, nil)
		stats.MonthlyExpense, _ = s.expenseRepo.Summary(&monthStart, nil, nil)
		stats.TotalExpense, _ = s.expenseRepo.Summary(nil, nil, nil)
		// Expense category breakdown for pie chart (this month)
		stats.ExpenseCategoryPie, _ = s.expenseRepo.CategoryTotals(&monthStart, nil)
	}

	// ── Cash in hand (always calculated, never stored) ────────────────────────
	stats.CashInHand = stats.TotalCollection - stats.TotalExpense

	// ── Monthly chart — last 12 months ────────────────────────────────────────
	stats.MonthlyChart = s.buildMonthlyChart(now)

	// ── Recent activity timeline ──────────────────────────────────────────────
	if s.activityRepo != nil {
		activities, _ := s.activityRepo.List(repositories.ActivityFilter{Limit: 20})
		stats.RecentActivities = activities
	} else {
		var activities []models.ActivityLog
		s.db.Preload("User").Where("activity_logs.deleted_at IS NULL").Order("created_at DESC").Limit(20).Find(&activities)
		stats.RecentActivities = activities
	}

	return stats, nil
}

func (s *dashboardService) buildMonthlyChart(now time.Time) []MonthlyPoint {
	points := make([]MonthlyPoint, 12)
	for i := 11; i >= 0; i-- {
		// Walk back i months from current month
		t := time.Date(now.Year(), now.Month()-time.Month(i), 1, 0, 0, 0, 0, now.Location())
		mStart := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, now.Location())
		mEnd := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, now.Location())

		var col, exp float64
		s.db.Raw(`SELECT COALESCE(SUM(amount), 0) FROM payment_records WHERE paid_at >= ? AND paid_at < ?`, mStart, mEnd).Scan(&col)
		if s.expenseRepo != nil {
			exp, _ = s.expenseRepo.Summary(&mStart, &mEnd, nil)
		}

		points[11-i] = MonthlyPoint{
			Year:       t.Year(),
			Month:      int(t.Month()),
			Label:      fmt.Sprintf("%s %d", monthNames[int(t.Month())], t.Year()),
			Collection: col,
			Expense:    exp,
			CashInHand: col - exp,
		}
	}
	return points
}
