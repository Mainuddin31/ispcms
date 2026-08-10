package com.ispcms.data.repository

import com.ispcms.data.api.ApiService
import com.ispcms.data.local.SecureStorage
import com.ispcms.data.models.LoginRequest
import com.ispcms.data.models.User
import com.ispcms.utils.PermissionHelper
import com.ispcms.utils.UiState
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class AuthRepository @Inject constructor(
    private val api: ApiService,
    private val storage: SecureStorage
) {
    val isLoggedIn: Boolean get() = storage.isLoggedIn
    val currentUser: User?  get() = storage.currentUser

    fun permissionHelper(): PermissionHelper =
        PermissionHelper(storage.currentUser?.role?.permissions ?: emptyList())

    suspend fun login(username: String, password: String): UiState<User> {
        return try {
            val res = api.login(LoginRequest(username, password))
            if (res.isSuccessful) {
                val body = res.body()!!
                storage.accessToken  = body.accessToken
                storage.refreshToken = body.refreshToken
                storage.currentUser  = body.user
                UiState.Success(body.user)
            } else {
                UiState.Error("Invalid username or password")
            }
        } catch (e: Exception) {
            UiState.Error("Could not connect to server. Check your connection.")
        }
    }

    suspend fun logout() {
        try { api.logout() } catch (_: Exception) {}
        storage.clearSession()
    }

    suspend fun refreshCurrentUser(): UiState<User> {
        return try {
            val res = api.me()
            if (res.isSuccessful) {
                val user = res.body()!!
                storage.currentUser = user
                UiState.Success(user)
            } else {
                UiState.Error("Failed to load user")
            }
        } catch (e: Exception) {
            UiState.Error(e.localizedMessage ?: "Network error")
        }
    }
}
