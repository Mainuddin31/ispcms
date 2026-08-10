package com.ispcms.ui.reports

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.ispcms.data.models.CollectionReportResponse
import com.ispcms.data.repository.ReportRepository
import com.ispcms.utils.UiState
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import java.util.Calendar
import javax.inject.Inject

@HiltViewModel
class ReportViewModel @Inject constructor(
    private val repo: ReportRepository
) : ViewModel() {

    private val now = Calendar.getInstance()

    private val _month = MutableStateFlow(now.get(Calendar.MONTH) + 1)
    val month: StateFlow<Int> = _month.asStateFlow()

    private val _year = MutableStateFlow(now.get(Calendar.YEAR))
    val year: StateFlow<Int> = _year.asStateFlow()

    private val _state = MutableStateFlow<UiState<CollectionReportResponse>>(UiState.Loading)
    val state: StateFlow<UiState<CollectionReportResponse>> = _state.asStateFlow()

    init { load() }

    fun load() {
        viewModelScope.launch {
            _state.value = UiState.Loading
            _state.value = repo.getCollectionReport(month = _month.value, year = _year.value)
        }
    }

    fun setMonth(month: Int, year: Int) {
        _month.value = month
        _year.value  = year
        load()
    }
}
