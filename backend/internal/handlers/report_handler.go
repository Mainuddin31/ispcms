package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/repositories"
	"github.com/ispcms/backend/pkg/utils"
)

type ReportHandler struct {
	reportRepo repositories.CollectionReportRepository
}

func NewReportHandler(reportRepo repositories.CollectionReportRepository) *ReportHandler {
	return &ReportHandler{reportRepo: reportRepo}
}

// ActiveUserCollection handles GET /api/v1/reports/active-user-collection
func (h *ReportHandler) ActiveUserCollection(c *fiber.Ctx) error {
	now := time.Now()

	// Period defaults to current billing month
	month := c.QueryInt("billing_month", int(now.Month()))
	year := c.QueryInt("billing_year", now.Year())
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 50)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	filter := repositories.CollectionReportFilter{
		BillingMonth:  month,
		BillingYear:   year,
		PaymentStatus: c.Query("payment_status"),
		Search:        c.Query("search"),
	}
	if id := c.Query("package_id"); id != "" {
		uid, err := uuid.Parse(id)
		if err == nil {
			filter.PackageID = &uid
		}
	}
	if id := c.Query("router_id"); id != "" {
		uid, err := uuid.Parse(id)
		if err == nil {
			filter.RouterID = &uid
		}
	}
	if id := c.Query("olt_id"); id != "" {
		uid, err := uuid.Parse(id)
		if err == nil {
			filter.OLTID = &uid
		}
	}
	if id := c.Query("pon_port_id"); id != "" {
		uid, err := uuid.Parse(id)
		if err == nil {
			filter.PONPortID = &uid
		}
	}
	if id := c.Query("collector_id"); id != "" {
		uid, err := uuid.Parse(id)
		if err == nil {
			filter.CollectorID = &uid
		}
	}

	// Run all queries in parallel via goroutines
	type rowsResult struct {
		rows  []repositories.CollectionRow
		total int64
		err   error
	}
	type summaryResult struct {
		summary *repositories.CollectionSummary
		err     error
	}
	type collectorResult struct {
		rows []repositories.CollectorSummaryRow
		err  error
	}
	type packageResult struct {
		rows []repositories.PackageSummaryRow
		err  error
	}
	type dailyResult struct {
		rows []repositories.DailyChartPoint
		err  error
	}

	rowsCh := make(chan rowsResult, 1)
	sumCh := make(chan summaryResult, 1)
	colCh := make(chan collectorResult, 1)
	pkgCh := make(chan packageResult, 1)
	dayCh := make(chan dailyResult, 1)

	go func() {
		rows, total, err := h.reportRepo.GetRows(filter, page, pageSize)
		rowsCh <- rowsResult{rows, total, err}
	}()
	go func() {
		s, err := h.reportRepo.GetSummary(filter)
		sumCh <- summaryResult{s, err}
	}()
	go func() {
		rows, err := h.reportRepo.GetCollectorSummary(filter)
		colCh <- collectorResult{rows, err}
	}()
	go func() {
		rows, err := h.reportRepo.GetPackageSummary(filter)
		pkgCh <- packageResult{rows, err}
	}()
	go func() {
		rows, err := h.reportRepo.GetDailyChart(filter)
		dayCh <- dailyResult{rows, err}
	}()

	rr := <-rowsCh
	sr := <-sumCh
	cr := <-colCh
	pr := <-pkgCh
	dr := <-dayCh

	for _, e := range []error{rr.err, sr.err, cr.err, pr.err, dr.err} {
		if e != nil {
			return utils.InternalError(c, e)
		}
	}

	totalPages := int(rr.total) / pageSize
	if int(rr.total)%pageSize != 0 {
		totalPages++
	}

	return utils.OK(c, fiber.Map{
		"summary":           sr.summary,
		"collector_summary": cr.rows,
		"package_summary":   pr.rows,
		"daily_chart":       dr.rows,
		"data":              rr.rows,
		"total":             rr.total,
		"page":              page,
		"page_size":         pageSize,
		"total_pages":       totalPages,
		"billing_month":     month,
		"billing_year":      year,
	})
}
