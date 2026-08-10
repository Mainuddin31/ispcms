package com.ispcms.ui.accounts

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.ispcms.data.models.InternetAccount
import com.ispcms.data.repository.AccountRepository
import com.ispcms.utils.UiState
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class AccountsViewModel @Inject constructor(
    private val repo: AccountRepository
) : ViewModel() {

    private val _state = MutableStateFlow<UiState<List<InternetAccount>>>(UiState.Loading)
    val state: StateFlow<UiState<List<InternetAccount>>> = _state.asStateFlow()

    private val _search = MutableStateFlow("")
    val search: StateFlow<String> = _search.asStateFlow()

    private var searchJob: Job? = null

    init { load() }

    fun load(search: String? = _search.value.takeIf { it.isNotBlank() }) {
        viewModelScope.launch {
            _state.value = UiState.Loading
            val result = repo.getAccounts(search = search)
            _state.value = when (result) {
                is UiState.Success -> UiState.Success(result.data.data)
                is UiState.Error   -> UiState.Error(result.message)
                else               -> UiState.Error("Unexpected state")
            }
        }
    }

    fun onSearch(query: String) {
        _search.value = query
        searchJob?.cancel()
        searchJob = viewModelScope.launch {
            delay(400)
            load(query.takeIf { it.isNotBlank() })
        }
    }
}
