package com.ispcms.data.models

import com.google.gson.annotations.SerializedName

// ── Auth ──────────────────────────────────────────────────────────────────────

data class LoginRequest(
    val username: String,
    val password: String
)

data class LoginResponse(
    @SerializedName("access_token")  val accessToken:  String,
    @SerializedName("refresh_token") val refreshToken: String,
    val user: User
)

data class RefreshRequest(
    @SerializedName("refresh_token") val refreshToken: String
)

data class RefreshResponse(
    @SerializedName("access_token")  val accessToken:  String,
    @SerializedName("refresh_token") val refreshToken: String? = null
)

// ── User / Role / Permission ───────────────────────────────────────────────

data class User(
    val id: String,
    val username: String,
    val email: String?,
    val name: String?,
    @SerializedName("is_active") val isActive: Boolean = true,
    val role: Role? = null
)

data class Role(
    val id: String,
    val name: String,
    val description: String?,
    val permissions: List<Permission> = emptyList()
)

data class Permission(
    val id: String,
    val module: String,
    val action: String
)

// ── Dashboard ─────────────────────────────────────────────────────────────────

data class DashboardStats(
    // Collections
    @SerializedName("today_collection")       val todayCollection:      Double = 0.0,
    @SerializedName("monthly_collection")     val monthlyCollection:    Double = 0.0,
    @SerializedName("last_month_collection")  val lastMonthCollection:  Double = 0.0,
    @SerializedName("total_outstanding_due")  val totalOutstandingDue:  Double = 0.0,
    // Expenses
    @SerializedName("today_expense")          val todayExpense:         Double = 0.0,
    @SerializedName("monthly_expense")        val monthlyExpense:       Double = 0.0,
    @SerializedName("last_month_expense")     val lastMonthExpense:     Double = 0.0,
    // Billing counts
    @SerializedName("total_packages")         val totalPackages:        Int    = 0,
    @SerializedName("total_subscriptions")    val totalSubscriptions:   Int    = 0,
    @SerializedName("total_bills")            val totalBills:           Int    = 0,
    @SerializedName("paid_bills")             val paidBills:            Int    = 0,
    @SerializedName("unpaid_bills")           val unpaidBills:          Int    = 0,
    // Accounts
    @SerializedName("total_accounts")         val totalAccounts:        Int    = 0,
    @SerializedName("online_accounts")        val onlineAccounts:       Int    = 0,
    // Routers
    @SerializedName("total_routers")          val totalRouters:         Int    = 0,
    @SerializedName("online_routers")         val onlineRouters:        Int    = 0,
    // Activity
    @SerializedName("recent_activity")        val recentActivity:       List<ActivityItem> = emptyList(),
    @SerializedName("recent_syncs")           val recentSyncs:          List<SyncLog>      = emptyList(),
    @SerializedName("monthly_chart")          val monthlyChart:         List<ChartEntry>   = emptyList()
)

data class ActivityItem(
    val id: String,
    @SerializedName("actor_name")  val actorName:  String?,
    val description: String,
    @SerializedName("created_at")  val createdAt:  String
)

data class SyncLog(
    val id: String,
    @SerializedName("router_name") val routerName: String,
    val status: String,
    @SerializedName("synced_at")   val syncedAt:   String,
    val message: String?
)

data class ChartEntry(
    val month: String,
    val collection: Double,
    val expense: Double
)

// ── Internet Accounts ──────────────────────────────────────────────────────

data class InternetAccount(
    val id: String,
    val username: String,
    val comment: String?,
    @SerializedName("is_online")    val isOnline:    Boolean = false,
    @SerializedName("current_ip")   val currentIp:   String?,
    val uptime: String?,
    val status: String = "active",
    @SerializedName("router_name")  val routerName:  String?,
    @SerializedName("package_name") val packageName: String?,
    @SerializedName("monthly_charge") val monthlyCharge: Double = 0.0
)

data class AccountListResponse(
    val data: List<InternetAccount>,
    val total: Int,
    val page: Int,
    @SerializedName("page_size") val pageSize: Int
)

// ── Bills ──────────────────────────────────────────────────────────────────

data class Bill(
    val id: String,
    @SerializedName("bill_number")  val billNumber:  String,
    val username: String,
    @SerializedName("package_name") val packageName: String?,
    @SerializedName("bill_month")   val billMonth:   String,
    val amount: Double,
    @SerializedName("paid_amount")  val paidAmount:  Double = 0.0,
    @SerializedName("due_amount")   val dueAmount:   Double = 0.0,
    val status: String,                       // paid | partial | unpaid
    @SerializedName("payment_method")  val paymentMethod:  String?,
    @SerializedName("receipt_number")  val receiptNumber:  String?,
    @SerializedName("collected_by")    val collectedBy:    String?,
    @SerializedName("last_paid_at")    val lastPaidAt:     String?,
    @SerializedName("internet_account_id") val internetAccountId: String
)

data class BillListResponse(
    val data: List<Bill>,
    val total: Int
)

data class AccountDueResponse(
    val bills: List<Bill>,
    @SerializedName("total_outstanding") val totalOutstanding: Double
)

data class CollectRequest(
    @SerializedName("internet_account_id") val internetAccountId: String,
    val amount: Double,
    @SerializedName("payment_method")  val paymentMethod:  String = "cash",
    @SerializedName("receipt_number")  val receiptNumber:  String? = null
)

data class CollectResponse(
    val message: String,
    @SerializedName("total_paid") val totalPaid: Double
)

// ── Collection Report ──────────────────────────────────────────────────────

data class CollectionReportRow(
    val username: String,
    @SerializedName("package_name")    val packageName:    String?,
    @SerializedName("bill_amount")     val billAmount:     Double,
    @SerializedName("paid_amount")     val paidAmount:     Double,
    @SerializedName("due_amount")      val dueAmount:      Double,
    val status: String,
    @SerializedName("last_payment")    val lastPayment:    String?,
    @SerializedName("collected_by")    val collectedBy:    String?
)

data class CollectionReportSummary(
    @SerializedName("total_active")    val totalActive:     Int,
    @SerializedName("total_collected") val totalCollected:  Int,
    @SerializedName("total_uncollected") val totalUncollected: Int,
    @SerializedName("collection_amount") val collectionAmount: Double,
    @SerializedName("total_bill")      val totalBill:       Double,
    @SerializedName("total_due")       val totalDue:        Double,
    @SerializedName("collection_rate") val collectionRate:  Double
)

data class CollectionReportResponse(
    val summary: CollectionReportSummary,
    val rows: List<CollectionReportRow>,
    @SerializedName("staff_cards") val staffCards: List<StaffCard>
)

data class StaffCard(
    @SerializedName("staff_name")  val staffName:  String,
    val amount: Double,
    val count: Int
)

// ── Expenses ──────────────────────────────────────────────────────────────

data class Expense(
    val id: String,
    val date: String,
    val category: String?,
    val amount: Double,
    val description: String?,
    val vendor: String?,
    @SerializedName("payment_method") val paymentMethod: String?,
    @SerializedName("reference_number") val referenceNumber: String?,
    @SerializedName("created_by_name") val createdByName: String?
)

data class ExpenseListResponse(
    val data: List<Expense>,
    val total: Int
)

data class CreateExpenseRequest(
    val date: String,
    @SerializedName("category_id") val categoryId: String?,
    val amount: Double,
    val description: String?,
    val vendor: String?,
    @SerializedName("payment_method") val paymentMethod: String = "cash",
    @SerializedName("reference_number") val referenceNumber: String?
)

data class ExpenseCategory(
    val id: String,
    val name: String,
    @SerializedName("is_active") val isActive: Boolean = true
)

// ── Generic API wrapper ────────────────────────────────────────────────────

data class ApiMessage(val message: String)

data class ApiError(
    val error: String?,
    val message: String?
) {
    fun readable() = error ?: message ?: "Unknown error"
}
