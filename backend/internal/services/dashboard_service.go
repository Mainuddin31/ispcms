package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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
	TodayExpense      float64 `json:"today_expense"`
	MonthlyExpense    float64 `json:"monthly_expense"`
	LastMonthExpense  float64 `json:"last_month_expense"`
	TotalExpense      float64 `json:"total_expense"`

	// Derived
	CashInHand float64 `json:"cash_in_hand"` // = TotalCollection - TotalExpense (never stored)

	// Charts
	MonthlyChart       []MonthlyPoint   `json:"monthly_chart"`
	ExpenseCategoryPie []models.CategoryTotal `json:"expense_category_pie"`

	RecentSyncLogs   []models.SyncLog     `json:"recent_sync_logs"`
	RecentActivities []models.ActivityLog `json:"recent_activities"`

	// Visiting
	TodayVisitsCount int64          `json:"today_visits_count"`
	TodayVisits      []models.Visit `json:"today_visits"`
}

type DashboardService interface {
	GetStats(prefixes []string, prefixRestricted bool, staffID *string) (*DashboardStats, error)
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
	visitRepo           repositories.VisitRepository
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
	visitRepo repositories.VisitRepository,
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
		visitRepo:           visitRepo,
		db:                  db,
	}
}

var monthNames = []string{"", "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

func (s *dashboardService) GetStats(prefixes []string, prefixRestricted bool, staffID *string) (*DashboardStats, error) {
	stats := &DashboardStats{}

	// Non-prefix-filtered stats (infrastructure / router / PPPoE)
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

	// Account counts — scoped to prefixes when prefix-restricted
	iaTotal, iaEnabled, iaDisabled, iaOnline, iaOffline, iaArchived, err :=
		s.internetAccountRepo.CountStats(prefixes, prefixRestricted)
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
	// Build optional prefix JOIN + WHERE clause for payment_records queries.
	// payment_records.internet_account_id → internet_accounts.username
	prJoin, prWhere, prArgs := s.buildPrefixFilter(prefixes, prefixRestricted, "payment_records", "pr_ia")
	mbJoin, mbWhere, mbArgs := s.buildPrefixFilter(prefixes, prefixRestricted, "monthly_bills", "mb_ia")

	s.db.Raw(
		`SELECT COALESCE(SUM(amount), 0) FROM payment_records `+prJoin+` WHERE paid_at >= ? `+prWhere,
		append([]interface{}{todayStart}, prArgs...)...,
	).Scan(&stats.TodayCollection)
	s.db.Raw(
		`SELECT COALESCE(SUM(amount), 0) FROM payment_records `+prJoin+` WHERE paid_at >= ? `+prWhere,
		append([]interface{}{monthStart}, prArgs...)...,
	).Scan(&stats.MonthlyCollection)
	s.db.Raw(
		`SELECT COALESCE(SUM(amount), 0) FROM payment_records `+prJoin+` WHERE paid_at >= ? AND paid_at <= ? `+prWhere,
		append([]interface{}{lastMonthStart, lastMonthEnd}, prArgs...)...,
	).Scan(&stats.LastMonthCollection)
	s.db.Raw(
		`SELECT COALESCE(SUM(amount), 0) FROM payment_records `+prJoin+` WHERE 1=1 `+prWhere,
		prArgs...,
	).Scan(&stats.TotalCollection)
	s.db.Raw(
		`SELECT COALESCE(SUM(due_amount), 0) FROM monthly_bills `+mbJoin+` WHERE status NOT IN ('paid', 'cancelled') `+mbWhere,
		mbArgs...,
	).Scan(&stats.TotalOutstandingDue)

	// Bill counts — scope to prefix accounts when restricted
	billQ := s.db.Model(&models.MonthlyBill{})
	billQPaid := s.db.Model(&models.MonthlyBill{})
	billQPending := s.db.Model(&models.MonthlyBill{})
	if prefixRestricted {
		if len(prefixes) == 0 {
			// No prefixes → zero results
			stats.TotalBillsGenerated = 0
			stats.BillsPaid = 0
			stats.BillsPending = 0
		} else {
			joinCond := "JOIN internet_accounts bill_ia ON bill_ia.id = monthly_bills.internet_account_id"
			prefixCond, pArgs := buildPrefixWhere("bill_ia", prefixes)
			billQ = billQ.Joins(joinCond).Where(prefixCond, pArgs...)
			billQPaid = billQPaid.Joins(joinCond).Where(prefixCond, pArgs...)
			billQPending = billQPending.Joins(joinCond).Where(prefixCond, pArgs...)
			billQ.Where("monthly_bills.status != 'cancelled'").Count(&stats.TotalBillsGenerated)
			billQPaid.Where("monthly_bills.status = 'paid'").Count(&stats.BillsPaid)
			billQPending.Where("monthly_bills.status IN ('pending', 'due', 'partial')").Count(&stats.BillsPending)
		}
	} else {
		billQ.Where("status != 'cancelled'").Count(&stats.TotalBillsGenerated)
		billQPaid.Where("status = 'paid'").Count(&stats.BillsPaid)
		billQPending.Where("status IN ('pending', 'due', 'partial')").Count(&stats.BillsPending)
	}

	// ── Expense stats ─────────────────────────────────────────────────────────
	if s.expenseRepo != nil {
		todayEnd := todayStart.Add(24 * time.Hour)
		stats.TodayExpense, _ = s.expenseRepo.Summary(&todayStart, &todayEnd, nil)
		stats.MonthlyExpense, _ = s.expenseRepo.Summary(&monthStart, nil, nil)
		stats.LastMonthExpense, _ = s.expenseRepo.Summary(&lastMonthStart, &lastMonthEnd, nil)
		stats.TotalExpense, _ = s.expenseRepo.Summary(nil, nil, nil)
		// Expense category breakdown for pie chart (this month)
		stats.ExpenseCategoryPie, _ = s.expenseRepo.CategoryTotals(&monthStart, nil)
	}

	// ── Today's visits ───────────────────────────────────────────────────────
	if s.visitRepo != nil {
		var visitStaffID *uuid.UUID
		if staffID != nil {
			if uid, err := uuid.Parse(*staffID); err == nil {
				visitStaffID = &uid
			}
		}
		todayVisits, _ := s.visitRepo.TodayVisits(visitStaffID)
		stats.TodayVisits = todayVisits
		stats.TodayVisitsCount = int64(len(todayVisits))
	}

	// ── Cash in hand (always calculated, never stored) ────────────────────────
	stats.CashInHand = stats.TotalCollection - stats.TotalExpense

	// ── Monthly chart — last 12 months ────────────────────────────────────────
	stats.MonthlyChart = s.buildMonthlyChart(now, prefixes, prefixRestricted)

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

func (s *dashboardService) buildMonthlyChart(now time.Time, prefixes []string, prefixRestricted bool) []MonthlyPoint {
	prJoin, prWhere, prArgs := s.buildPrefixFilter(prefixes, prefixRestricted, "payment_records", "chart_ia")
	points := make([]MonthlyPoint, 12)
	for i := 11; i >= 0; i-- {
		t := time.Date(now.Year(), now.Month()-time.Month(i), 1, 0, 0, 0, 0, now.Location())
		mStart := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, now.Location())
		mEnd := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, now.Location())

		var col, exp float64
		args := append([]interface{}{mStart, mEnd}, prArgs...)
		s.db.Raw(
			`SELECT COALESCE(SUM(amount), 0) FROM payment_records `+prJoin+` WHERE paid_at >= ? AND paid_at < ? `+prWhere,
			args...,
		).Scan(&col)
		// Expenses are company-wide (not per-account) — always unfiltered
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

// buildPrefixFilter returns the JOIN clause and WHERE fragment (with leading space + AND)
// for scoping payment_records or monthly_bills to accounts whose username matches a prefix.
// When prefixRestricted=true and prefixes is empty, returns an impossible condition.
// tableAlias is the alias for the internet_accounts join table.
func (s *dashboardService) buildPrefixFilter(prefixes []string, prefixRestricted bool, baseTable, tableAlias string) (joinSQL, whereSQL string, args []interface{}) {
	if !prefixRestricted {
		return "", "", nil
	}
	if len(prefixes) == 0 {
		// No allowed prefixes → impossible condition, return nothing
		return "", "AND 1=0", nil
	}
	joinSQL = fmt.Sprintf("JOIN internet_accounts %s ON %s.id = %s.internet_account_id", tableAlias, tableAlias, baseTable)
	cond, a := buildPrefixWhere(tableAlias, prefixes)
	whereSQL = "AND (" + cond + ")"
	args = a
	return
}

// buildPrefixWhere builds the ILIKE OR condition for prefix matching.
func buildPrefixWhere(alias string, prefixes []string) (string, []interface{}) {
	parts := make([]string, len(prefixes))
	args := make([]interface{}, len(prefixes))
	for i, p := range prefixes {
		parts[i] = alias + ".username ILIKE ?"
		args[i] = p + "%"
	}
	return strings.Join(parts, " OR "), args
}
