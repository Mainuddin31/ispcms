package com.ispcms.data.api

import com.ispcms.data.local.SecureStorage
import okhttp3.Interceptor
import okhttp3.Response
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Attaches the stored access token to every outgoing request as a Bearer header.
 */
@Singleton
class AuthInterceptor @Inject constructor(
    private val storage: SecureStorage
) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val original = chain.request()
        val token = storage.accessToken ?: return chain.proceed(original)
        val request = original.newBuilder()
            .header("Authorization", "Bearer $token")
            .build()
        return chain.proceed(request)
    }
}
