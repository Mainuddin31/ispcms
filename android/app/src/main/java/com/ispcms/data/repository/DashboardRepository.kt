package com.ispcms.data.repository

import com.ispcms.data.api.ApiService
import com.ispcms.data.models.DashboardStats
import com.ispcms.utils.UiState
import com.ispcms.utils.safeApiCall
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class DashboardRepository @Inject constructor(private val api: ApiService) {
    suspend fun getStats(): UiState<DashboardStats> = safeApiCall { api.dashboardStats() }
}
