package com.ispcms.ui.main

import androidx.lifecycle.ViewModel
import com.ispcms.data.repository.AuthRepository
import com.ispcms.utils.PermissionHelper
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import javax.inject.Inject

@HiltViewModel
class MainViewModel @Inject constructor(
    private val authRepo: AuthRepository
) : ViewModel() {

    private val _isLoggedIn = MutableStateFlow(authRepo.isLoggedIn)
    val isLoggedIn: StateFlow<Boolean> = _isLoggedIn.asStateFlow()

    val currentUser get() = authRepo.currentUser

    val permissions: PermissionHelper
        get() = authRepo.permissionHelper()

    fun refreshLoginState() {
        _isLoggedIn.value = authRepo.isLoggedIn
    }
}
