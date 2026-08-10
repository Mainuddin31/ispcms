package com.ispcms.ui.auth

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.ispcms.data.models.User
import com.ispcms.data.repository.AuthRepository
import com.ispcms.utils.UiState
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class LoginViewModel @Inject constructor(
    private val authRepo: AuthRepository
) : ViewModel() {

    private val _loginState = MutableStateFlow<UiState<User>>(UiState.Idle)
    val loginState: StateFlow<UiState<User>> = _loginState.asStateFlow()

    fun login(username: String, password: String) {
        if (username.isBlank() || password.isBlank()) {
            _loginState.value = UiState.Error("Username and password are required")
            return
        }
        viewModelScope.launch {
            _loginState.value = UiState.Loading
            _loginState.value = authRepo.login(username.trim(), password)
        }
    }

    fun resetState() {
        _loginState.value = UiState.Idle
    }
}
