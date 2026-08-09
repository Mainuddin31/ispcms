package repositories

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CollectionReportFilter holds all filter parameters for the collection report.
type CollectionReportFilter struct {
	BillingMonth  int
	BillingYear   int
	PaymentStatus string // all | paid | partial | unpaid | no_bill
	PackageID     *uuid.UUID
	RouterID      *uuid.UUID
	OLTID         *uuid.UUID
	PONPortID     *uuid.UUID
	CollectorID   *uuid.UUID
	Search        string
}

// CollectionRow is one row in the collection detail table.
type CollectionRow struct {
	AccountID      string  `json:"account_id"`
	CustomerName   string  `json:"customer_name"`
	Username       string  `json:"username"`
	RouterName     string  `json:"router_name"`
	RouterID       string  `json:"router_id"`
	PackageName    string  `json:"package_name"`
	PackageID      string  `json:"package_id"`
	MonthlyCharge  float64 `json:"monthly_charge"`
	BillID         string  `json:"bill_id"`
	BillNumber     string  `json:"bill_number"`
	TotalAmount    float64 `json:"total_amount"`
	PaidAmount     float64 `json:"paid_amount"`
	DueAmount      float64 `json:"due_amount"`
	BillStatus     string  `json:"bill_status"`
	PaymentStatus  string  `json:"payment_status"` // paid | partial | unpaid | no_bill
	LastPaymentAt  *time.Time `json:"last_payment_at"`
	CollectorName  string  `json:"collector_name"`
	CollectorID    string  `json:"collector_id"`
	OLTName        string  `json:"olt_name"`
	OLTID          string  `json:"olt_id"`
	PONPortLabel   string  `json:"pon_port_label"`
	PONPortID      string  `json:"pon_port_id"`
	OnuID          string  `json:"onu_id"`
	OnuMAC         string  `json:"onu_mac"`
	OnuStatus      string  `json:"onu_status"`
}

// CollectionSummary holds the summary statistics.
type CollectionSummary struct {
	ActiveClients      int64   `json:"active_clients"`
	CollectedClients   int64   `json:"collected_clients"`
	UncollectedClients int64   `json:"uncollected_clients"`
	TotalBill          float64 `json:"total_bill"`
	CollectionAmount   float64 `json:"collection_amount"`
	TotalDue           float64 `json:"total_due"`
	CollectionRate     float64 `json:"collection_rate"`
}

// CollectorSummaryRow summarises collection per collector.
type CollectorSummaryRow struct {
	CollectorID   string  `json:"collector_id"`
	CollectorName string  `json:"collector_name"`
	ClientCount   int64   `json:"client_count"`
	Collection    float64 `json:"collection"`
}

// PackageSummaryRow summarises collection per package.
type PackageSummaryRow struct {
	PackageID   string  `json:"package_id"`
	PackageName string  `json:"package_name"`
	ClientCount int64   `json:"client_count"`
	Collection  float64 `json:"collection"`
}

// DailyChartPoint is one day of collection.
type DailyChartPoint struct {
	Date       string  `json:"date"`        // "2026-07-01"
	Label      string  `json:"label"`       // "Jul 01"
	Collection float64 `json:"collection"`
}

// CollectionReportRepository provides all queries for the collection report.
type CollectionReportRepository interface {
	GetRows(filter CollectionReportFilter, page, pageSize int) ([]CollectionRow, int64, error)
	GetSummary(filter CollectionReportFilter) (*CollectionSummary, error)
	GetCollectorSummary(filter CollectionReportFilter) ([]CollectorSummaryRow, error)
	GetPackageSummary(filter CollectionReportFilter) ([]PackageSummaryRow, error)
	GetDailyChart(filter CollectionReportFilter) ([]DailyChartPoint, error)
}

type collectionReportRepository struct{ db *gorm.DB }

func NewCollectionReportRepository(db *gorm.DB) CollectionReportRepository {
	return &collectionReportRepository{db: db}
}

// baseWhere builds the WHERE clause and argument list shared by all queries.
// It returns the WHERE SQL string (starting with "WHERE") and args slice.
func (r *collectionReportRepository) baseWhere(f CollectionReportFilter) (string, []interface{}) {
	clauses := []string{"ia.archived_at IS NULL", "ia.disabled = false"}
	args := []interface{}{}

	// OLT / PON port filtering (via ONU link)
	if f.OLTID != nil {
		clauses = append(clauses, "onu.olt_id = ?")
		args = append(args, *f.OLTID)
	}
	if f.PONPortID != nil {
		clauses = append(clauses, "onu.pon_port_id = ?")
		args = append(args, *f.PONPortID)
	}
	if f.RouterID != nil {
		clauses = append(clauses, "ia.router_id = ?")
		args = append(args, *f.RouterID)
	}
	if f.PackageID != nil {
		clauses = append(clauses, "cs.package_id = ?")
		args = append(args, *f.PackageID)
	}
	if f.CollectorID != nil {
		clauses = append(clauses, `EXISTS (
			SELECT 1 FROM payment_records pr2
			WHERE pr2.bill_id = mb.id AND pr2.received_by_id = ?
		)`)
		args = append(args, *f.CollectorID)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		clauses = append(clauses, `(ia.username ILIKE ? OR ia.comment ILIKE ? OR mb.bill_number ILIKE ?)`)
		args = append(args, like, like, like)
	}

	// payment_status filter applied after join
	switch f.PaymentStatus {
	case "paid":
		clauses = append(clauses, "mb.id IS NOT NULL AND mb.paid_amount >= mb.total_amount")
	case "partial":
		clauses = append(clauses, "mb.id IS NOT NULL AND mb.paid_amount > 0 AND mb.paid_amount < mb.total_amount")
	case "unpaid":
		clauses = append(clauses, "mb.id IS NOT NULL AND mb.paid_amount = 0")
	case "no_bill":
		clauses = append(clauses, "mb.id IS NULL")
	}

	return "WHERE " + strings.Join(clauses, " AND "), args
}

// baseFromJoins returns the FROM + JOIN SQL (without WHERE) for the collection
// detail query. The billing_month and billing_year are injected here.
func baseFromJoins(month, year int) (string, []interface{}) {
	sql := `
FROM internet_accounts ia
LEFT JOIN routers r ON r.id = ia.router_id
LEFT JOIN customer_subscriptions cs ON cs.internet_account_id = ia.id AND cs.is_active = true
LEFT JOIN packages p ON p.id = cs.package_id
LEFT JOIN monthly_bills mb
    ON mb.internet_account_id = ia.id
   AND mb.billing_month = ?
   AND mb.billing_year  = ?
   AND mb.status != 'cancelled'
LEFT JOIN onus onu ON onu.internet_account_id = ia.id AND onu.archived_at IS NULL
LEFT JOIN pon_ports pp ON pp.id = onu.pon_port_id
LEFT JOIN olts olt ON olt.id = onu.olt_id
`
	return sql, []interface{}{month, year}
}

func (r *collectionReportRepository) GetRows(
	filter CollectionReportFilter, page, pageSize int,
) ([]CollectionRow, int64, error) {
	fromSQL, fromArgs := baseFromJoins(filter.BillingMonth, filter.BillingYear)
	whereSQL, whereArgs := r.baseWhere(filter)
	args := append(fromArgs, whereArgs...)

	// count
	var total int64
	countSQL := "SELECT COUNT(*) " + fromSQL + whereSQL
	if err := r.db.Raw(countSQL, args...).Scan(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}

	// data
	selectSQL := `
SELECT
    ia.id::text                                   AS account_id,
    COALESCE(NULLIF(ia.comment,''), ia.username)  AS customer_name,
    ia.username,
    COALESCE(r.name,'')                           AS router_name,
    COALESCE(r.id::text,'')                       AS router_id,
    COALESCE(p.display_name,'')                   AS package_name,
    COALESCE(cs.package_id::text,'')              AS package_id,
    COALESCE(cs.monthly_price,0)                  AS monthly_charge,
    COALESCE(mb.id::text,'')                      AS bill_id,
    COALESCE(mb.bill_number,'')                   AS bill_number,
    COALESCE(mb.total_amount,0)                   AS total_amount,
    COALESCE(mb.paid_amount,0)                    AS paid_amount,
    COALESCE(mb.due_amount,0)                     AS due_amount,
    COALESCE(mb.status,'no_bill')                 AS bill_status,
    CASE
        WHEN mb.id IS NULL              THEN 'no_bill'
        WHEN mb.paid_amount >= mb.total_amount AND mb.total_amount > 0 THEN 'paid'
        WHEN mb.paid_amount > 0         THEN 'partial'
        ELSE                                 'unpaid'
    END                                           AS payment_status,
    (SELECT MAX(pr.paid_at)
     FROM payment_records pr WHERE pr.bill_id = mb.id)       AS last_payment_at,
    COALESCE((
        SELECT u.full_name FROM payment_records pr
        LEFT JOIN users u ON u.id = pr.received_by_id
        WHERE pr.bill_id = mb.id ORDER BY pr.paid_at DESC LIMIT 1
    ),'')                                         AS collector_name,
    COALESCE((
        SELECT pr.received_by_id::text FROM payment_records pr
        WHERE pr.bill_id = mb.id ORDER BY pr.paid_at DESC LIMIT 1
    ),'')                                         AS collector_id,
    COALESCE(olt.name,'')                         AS olt_name,
    COALESCE(olt.id::text,'')                     AS olt_id,
    COALESCE(pp.label,'')                         AS pon_port_label,
    COALESCE(pp.id::text,'')                      AS pon_port_id,
    COALESCE(onu.onu_id,'')                       AS onu_id,
    COALESCE(onu.mac_address,'')                  AS onu_mac,
    COALESCE(onu.status,'')                       AS onu_status
`
	dataSQL := selectSQL + fromSQL + whereSQL + " ORDER BY ia.username ASC LIMIT ? OFFSET ?"
	dataArgs := append(args, pageSize, (page-1)*pageSize)

	var rows []CollectionRow
	if err := r.db.Raw(dataSQL, dataArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("rows: %w", err)
	}
	return rows, total, nil
}

func (r *collectionReportRepository) GetSummary(filter CollectionReportFilter) (*CollectionSummary, error) {
	fromSQL, fromArgs := baseFromJoins(filter.BillingMonth, filter.BillingYear)
	whereSQL, whereArgs := r.baseWhere(filter)
	args := append(fromArgs, whereArgs...)

	// We always want all statuses for summary, so strip payment_status from where
	// by re-building without it:
	filterForSummary := filter
	filterForSummary.PaymentStatus = ""
	whereAllSQL, whereAllArgs := r.baseWhere(filterForSummary)
	allArgs := append(fromArgs, whereAllArgs...)

	summarySQL := `
SELECT
    COUNT(ia.id)                                                           AS active_clients,
    COUNT(CASE WHEN mb.paid_amount > 0 THEN 1 END)                        AS collected_clients,
    COALESCE(SUM(mb.total_amount),0)                                       AS total_bill,
    COALESCE(SUM(mb.paid_amount),0)                                        AS collection_amount,
    COALESCE(SUM(mb.due_amount),0)                                         AS total_due
` + fromSQL + whereAllSQL

	type rawSummary struct {
		ActiveClients    int64   `gorm:"column:active_clients"`
		CollectedClients int64   `gorm:"column:collected_clients"`
		TotalBill        float64 `gorm:"column:total_bill"`
		CollectionAmount float64 `gorm:"column:collection_amount"`
		TotalDue         float64 `gorm:"column:total_due"`
	}
	var rs rawSummary
	if err := r.db.Raw(summarySQL, allArgs...).Scan(&rs).Error; err != nil {
		return nil, fmt.Errorf("summary: %w", err)
	}

	_ = args // suppress unused warning
	s := &CollectionSummary{
		ActiveClients:      rs.ActiveClients,
		CollectedClients:   rs.CollectedClients,
		UncollectedClients: rs.ActiveClients - rs.CollectedClients,
		TotalBill:          rs.TotalBill,
		CollectionAmount:   rs.CollectionAmount,
		TotalDue:           rs.TotalDue,
	}
	if rs.ActiveClients > 0 {
		s.CollectionRate = float64(rs.CollectedClients) / float64(rs.ActiveClients) * 100
	}
	return s, nil
}

func (r *collectionReportRepository) GetCollectorSummary(filter CollectionReportFilter) ([]CollectorSummaryRow, error) {
	sql := `
SELECT
    COALESCE(u.id::text,'unknown')   AS collector_id,
    COALESCE(u.full_name,'Unknown')  AS collector_name,
    COUNT(DISTINCT pr.bill_id)       AS client_count,
    COALESCE(SUM(pr.amount),0)       AS collection
FROM payment_records pr
LEFT JOIN users u ON u.id = pr.received_by_id
LEFT JOIN monthly_bills mb ON mb.id = pr.bill_id
LEFT JOIN internet_accounts ia ON ia.id = pr.internet_account_id
WHERE mb.billing_month = ? AND mb.billing_year = ? AND mb.status != 'cancelled'
  AND ia.archived_at IS NULL AND ia.disabled = false
GROUP BY u.id, u.full_name
ORDER BY collection DESC
`
	var rows []CollectorSummaryRow
	if err := r.db.Raw(sql, filter.BillingMonth, filter.BillingYear).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("collector summary: %w", err)
	}
	return rows, nil
}

func (r *collectionReportRepository) GetPackageSummary(filter CollectionReportFilter) ([]PackageSummaryRow, error) {
	sql := `
SELECT
    COALESCE(p.id::text,'')      AS package_id,
    COALESCE(p.display_name,'No Package') AS package_name,
    COUNT(DISTINCT ia.id)        AS client_count,
    COALESCE(SUM(mb.paid_amount),0) AS collection
FROM internet_accounts ia
LEFT JOIN customer_subscriptions cs ON cs.internet_account_id = ia.id AND cs.is_active = true
LEFT JOIN packages p ON p.id = cs.package_id
LEFT JOIN monthly_bills mb
    ON mb.internet_account_id = ia.id
   AND mb.billing_month = ? AND mb.billing_year = ?
   AND mb.status != 'cancelled'
WHERE ia.archived_at IS NULL AND ia.disabled = false
GROUP BY p.id, p.display_name
ORDER BY collection DESC
`
	var rows []PackageSummaryRow
	if err := r.db.Raw(sql, filter.BillingMonth, filter.BillingYear).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("package summary: %w", err)
	}
	return rows, nil
}

func (r *collectionReportRepository) GetDailyChart(filter CollectionReportFilter) ([]DailyChartPoint, error) {
	sql := `
SELECT
    TO_CHAR(pr.paid_at, 'YYYY-MM-DD')  AS date,
    TO_CHAR(pr.paid_at, 'Mon DD')      AS label,
    COALESCE(SUM(pr.amount),0)          AS collection
FROM payment_records pr
LEFT JOIN monthly_bills mb ON mb.id = pr.bill_id
WHERE mb.billing_month = ? AND mb.billing_year = ? AND mb.status != 'cancelled'
GROUP BY TO_CHAR(pr.paid_at, 'YYYY-MM-DD'), TO_CHAR(pr.paid_at, 'Mon DD')
ORDER BY date ASC
`
	var rows []DailyChartPoint
	if err := r.db.Raw(sql, filter.BillingMonth, filter.BillingYear).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("daily chart: %w", err)
	}
	return rows, nil
}
