package com.ispcms.data.repository

import com.ispcms.data.api.ApiService
import com.ispcms.data.models.CollectionReportResponse
import com.ispcms.utils.UiState
import com.ispcms.utils.safeApiCall
import java.util.Calendar
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class ReportRepository @Inject constructor(private val api: ApiService) {

    suspend fun getCollectionReport(
        month:  Int    = Calendar.getInstance().get(Calendar.MONTH) + 1,
        year:   Int    = Calendar.getInstance().get(Calendar.YEAR),
        status: String? = null,
        search: String? = null
    ): UiState<CollectionReportResponse> = safeApiCall {
        api.collectionReport(
            month  = month,
            year   = year,
            status = status,
            search = search.takeIf { !it.isNullOrBlank() }
        )
    }
}
