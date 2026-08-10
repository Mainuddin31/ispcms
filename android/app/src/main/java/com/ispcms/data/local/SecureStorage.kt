package com.ispcms.data.local

import android.content.Context
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import com.google.gson.Gson
import com.ispcms.data.models.User
import dagger.hilt.android.qualifiers.ApplicationContext
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Wraps EncryptedSharedPreferences backed by an AES-256-GCM MasterKey stored in the
 * Android Keystore. Tokens are encrypted at rest and never written to plain storage.
 */
@Singleton
class SecureStorage @Inject constructor(
    @ApplicationContext private val context: Context
) {
    private val gson = Gson()

    private val prefs by lazy {
        val masterKey = MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
        EncryptedSharedPreferences.create(
            context,
            "ispcms_secure_prefs",
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
        )
    }

    // ── Tokens ─────────────────────────────────────────────────────────────────

    var accessToken: String?
        get()      = prefs.getString(KEY_ACCESS_TOKEN, null)
        set(value) = prefs.edit().putString(KEY_ACCESS_TOKEN, value).apply()

    var refreshToken: String?
        get()      = prefs.getString(KEY_REFRESH_TOKEN, null)
        set(value) = prefs.edit().putString(KEY_REFRESH_TOKEN, value).apply()

    // ── Current user (cached after login) ─────────────────────────────────────

    var currentUser: User?
        get() = prefs.getString(KEY_USER, null)?.let { gson.fromJson(it, User::class.java) }
        set(value) {
            if (value == null) prefs.edit().remove(KEY_USER).apply()
            else prefs.edit().putString(KEY_USER, gson.toJson(value)).apply()
        }

    // ── Helpers ───────────────────────────────────────────────────────────────

    val isLoggedIn: Boolean get() = accessToken != null

    fun clearSession() {
        prefs.edit()
            .remove(KEY_ACCESS_TOKEN)
            .remove(KEY_REFRESH_TOKEN)
            .remove(KEY_USER)
            .apply()
    }

    companion object {
        private const val KEY_ACCESS_TOKEN  = "access_token"
        private const val KEY_REFRESH_TOKEN = "refresh_token"
        private const val KEY_USER          = "current_user"
    }
}
