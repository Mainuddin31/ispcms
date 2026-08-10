package com.ispcms.data.repository

import com.ispcms.data.api.ApiService
import com.ispcms.data.models.*
import com.ispcms.utils.UiState
import com.ispcms.utils.safeApiCall
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class BillRepository @Inject constructor(private val api: ApiService) {

    suspend fun getBills(
        page: Int = 1,
        search: String? = null,
        status: String? = null,
        month: String? = null
    ): UiState<BillListResponse> = safeApiCall {
        api.bills(
            page   = page,
            search = search.takeIf { !it.isNullOrBlank() },
            status = status,
            month  = month
        )
    }

    suspend fun getAccountDue(accountId: String): UiState<AccountDueResponse> =
        safeApiCall { api.accountDue(accountId) }

    suspend fun collectPayment(
        accountId:     String,
        amount:        Double,
        paymentMethod: String  = "cash",
        receiptNumber: String? = null
    ): UiState<CollectResponse> = safeApiCall {
        api.collectPayment(
            CollectRequest(
                internetAccountId = accountId,
                amount            = amount,
                paymentMethod     = paymentMethod,
                receiptNumber     = receiptNumber.takeIf { !it.isNullOrBlank() }
            )
        )
    }
}
