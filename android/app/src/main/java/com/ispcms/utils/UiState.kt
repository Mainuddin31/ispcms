package com.ispcms.utils

/**
 * Generic sealed UI state used in every ViewModel.
 */
sealed class UiState<out T> {
    data object Loading : UiState<Nothing>()
    data class  Success<T>(val data: T) : UiState<T>()
    data class  Error(val message: String) : UiState<Nothing>()
    data object Idle : UiState<Nothing>()
}

/**
 * Safely execute a suspend API call and wrap the result in UiState.
 */
suspend fun <T> safeApiCall(block: suspend () -> retrofit2.Response<T>): UiState<T> {
    return try {
        val response = block()
        if (response.isSuccessful) {
            val body = response.body()
            if (body != null) UiState.Success(body)
            else UiState.Error("Empty response")
        } else {
            UiState.Error("Error ${response.code()}: ${response.message()}")
        }
    } catch (e: Exception) {
        UiState.Error(e.localizedMessage ?: "Network error")
    }
}
