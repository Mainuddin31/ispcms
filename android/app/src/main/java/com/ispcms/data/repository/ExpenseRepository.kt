package com.ispcms.data.repository

import com.ispcms.data.api.ApiService
import com.ispcms.data.models.*
import com.ispcms.utils.UiState
import com.ispcms.utils.safeApiCall
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class ExpenseRepository @Inject constructor(private val api: ApiService) {

    suspend fun getExpenses(page: Int = 1): UiState<ExpenseListResponse> =
        safeApiCall { api.expenses(page = page) }

    suspend fun getCategories(): UiState<List<ExpenseCategory>> =
        safeApiCall { api.expenseCategories() }

    suspend fun createExpense(request: CreateExpenseRequest): UiState<Expense> =
        safeApiCall { api.createExpense(request) }
}
