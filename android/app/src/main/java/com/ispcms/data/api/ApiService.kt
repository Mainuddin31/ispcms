package com.ispcms.data.api

import com.ispcms.data.models.*
import retrofit2.Response
import retrofit2.http.*

interface ApiService {

    // ── Auth ──────────────────────────────────────────────────────────────────

    @POST("auth/login")
    suspend fun login(@Body request: LoginRequest): Response<LoginResponse>

    @POST("auth/refresh")
    suspend fun refresh(@Body request: RefreshRequest): Response<RefreshResponse>

    @POST("auth/logout")
    suspend fun logout(): Response<ApiMessage>

    @GET("auth/me")
    suspend fun me(): Response<User>

    // ── Dashboard ─────────────────────────────────────────────────────────────

    @GET("dashboard/stats")
    suspend fun dashboardStats(): Response<DashboardStats>

    // ── Internet Accounts ──────────────────────────────────────────────────

    @GET("internet-accounts")
    suspend fun accounts(
        @Query("page")     page:   Int    = 1,
        @Query("limit")    limit:  Int    = 30,
        @Query("search")   search: String? = null,
        @Query("status")   status: String? = null
    ): Response<AccountListResponse>

    // ── Bills ──────────────────────────────────────────────────────────────

    @GET("bills")
    suspend fun bills(
        @Query("page")   page:   Int     = 1,
        @Query("limit")  limit:  Int     = 30,
        @Query("search") search: String? = null,
        @Query("status") status: String? = null,
        @Query("month")  month:  String? = null
    ): Response<BillListResponse>

    @GET("bills/account-due")
    suspend fun accountDue(
        @Query("internet_account_id") accountId: String
    ): Response<AccountDueResponse>

    @POST("bills/collect")
    suspend fun collectPayment(@Body request: CollectRequest): Response<CollectResponse>

    // ── Collection Report ──────────────────────────────────────────────────

    @GET("reports/active-user-collection")
    suspend fun collectionReport(
        @Query("month")   month:  Int,
        @Query("year")    year:   Int,
        @Query("status")  status: String? = null,
        @Query("search")  search: String? = null
    ): Response<CollectionReportResponse>

    // ── Expenses ──────────────────────────────────────────────────────────

    @GET("expenses")
    suspend fun expenses(
        @Query("page")  page:  Int = 1,
        @Query("limit") limit: Int = 30
    ): Response<ExpenseListResponse>

    @POST("expenses")
    suspend fun createExpense(@Body request: CreateExpenseRequest): Response<Expense>

    @GET("expense-categories")
    suspend fun expenseCategories(): Response<List<ExpenseCategory>>
}
