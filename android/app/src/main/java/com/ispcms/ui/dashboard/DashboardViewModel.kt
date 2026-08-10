package com.ispcms.ui.dashboard

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.ispcms.data.models.DashboardStats
import com.ispcms.data.repository.DashboardRepository
import com.ispcms.utils.UiState
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class DashboardViewModel @Inject constructor(
    private val repo: DashboardRepository
) : ViewModel() {

    private val _stats = MutableStateFlow<UiState<DashboardStats>>(UiState.Loading)
    val stats: StateFlow<UiState<DashboardStats>> = _stats.asStateFlow()

    init { load() }

    fun load() {
        viewModelScope.launch {
            _stats.value = UiState.Loading
            _stats.value = repo.getStats()
        }
    }
}
