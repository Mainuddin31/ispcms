package com.ispcms.data.repository

import com.ispcms.data.api.ApiService
import com.ispcms.data.models.AccountListResponse
import com.ispcms.utils.UiState
import com.ispcms.utils.safeApiCall
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class AccountRepository @Inject constructor(private val api: ApiService) {

    suspend fun getAccounts(
        page: Int = 1,
        search: String? = null,
        status: String? = null
    ): UiState<AccountListResponse> = safeApiCall {
        api.accounts(page = page, search = search.takeIf { !it.isNullOrBlank() }, status = status)
    }
}
