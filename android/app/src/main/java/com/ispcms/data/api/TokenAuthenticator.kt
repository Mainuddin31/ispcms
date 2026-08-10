package com.ispcms.data.api

import com.ispcms.data.local.SecureStorage
import com.ispcms.data.models.RefreshRequest
import kotlinx.coroutines.runBlocking
import okhttp3.Authenticator
import okhttp3.Request
import okhttp3.Response
import okhttp3.Route
import retrofit2.Retrofit
import javax.inject.Inject
import javax.inject.Provider
import javax.inject.Singleton

/**
 * Called automatically by OkHttp when the server returns 401.
 *
 * Flow:
 *  1. Try to exchange the stored refresh token for a new access token.
 *  2. On success: save new tokens and retry the original request.
 *  3. On failure: clear session so the app redirects to Login.
 *
 * Uses Provider<ApiService> (lazy) to avoid a circular Hilt dependency between
 * OkHttpClient → TokenAuthenticator → ApiService → OkHttpClient.
 */
@Singleton
class TokenAuthenticator @Inject constructor(
    private val storage: SecureStorage,
    private val apiServiceProvider: Provider<ApiService>
) : Authenticator {

    override fun authenticate(route: Route?, response: Response): Request? {
        // If we already retried once, give up and force logout
        if (response.request.header("X-Retry-Auth") != null) {
            storage.clearSession()
            return null
        }

        val refresh = storage.refreshToken ?: run {
            storage.clearSession()
            return null
        }

        val newAccessToken = runBlocking {
            try {
                val res = apiServiceProvider.get()
                    .refresh(RefreshRequest(refresh))
                if (res.isSuccessful) {
                    val body = res.body()!!
                    storage.accessToken = body.accessToken
                    body.refreshToken?.let { storage.refreshToken = it }
                    body.accessToken
                } else {
                    null
                }
            } catch (e: Exception) {
                null
            }
        }

        if (newAccessToken == null) {
            storage.clearSession()
            return null
        }

        return response.request.newBuilder()
            .header("Authorization", "Bearer $newAccessToken")
            .header("X-Retry-Auth", "true")
            .build()
    }
}
