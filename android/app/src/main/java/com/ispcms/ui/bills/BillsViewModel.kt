package com.ispcms.ui.bills

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.ispcms.data.models.AccountDueResponse
import com.ispcms.data.models.Bill
import com.ispcms.data.models.CollectResponse
import com.ispcms.data.repository.BillRepository
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
class BillsViewModel @Inject constructor(
    private val repo: BillRepository
) : ViewModel() {

    private val _bills  = MutableStateFlow<UiState<List<Bill>>>(UiState.Loading)
    val bills: StateFlow<UiState<List<Bill>>> = _bills.asStateFlow()

    private val _search = MutableStateFlow("")
    val search: StateFlow<String> = _search.asStateFlow()

    // Collect payment dialog state
    private val _dueState    = MutableStateFlow<UiState<AccountDueResponse>>(UiState.Idle)
    val dueState: StateFlow<UiState<AccountDueResponse>> = _dueState.asStateFlow()

    private val _collectState = MutableStateFlow<UiState<CollectResponse>>(UiState.Idle)
    val collectState: StateFlow<UiState<CollectResponse>> = _collectState.asStateFlow()

    private var searchJob: Job? = null

    init { load() }

    fun load(search: String? = _search.value.takeIf { it.isNotBlank() }) {
        viewModelScope.launch {
            _bills.value = UiState.Loading
            val res = repo.getBills(search = search)
            _bills.value = when (res) {
                is UiState.Success -> UiState.Success(res.data.data)
                is UiState.Error   -> UiState.Error(res.message)
                else               -> UiState.Error("Unexpected")
            }
        }
    }

    fun onSearch(q: String) {
        _search.value = q
        searchJob?.cancel()
        searchJob = viewModelScope.launch {
            delay(400)
            load(q.takeIf { it.isNotBlank() })
        }
    }

    fun loadDue(accountId: String) {
        viewModelScope.launch {
            _dueState.value = UiState.Loading
            _dueState.value = repo.getAccountDue(accountId)
        }
    }

    fun collect(accountId: String, amount: Double, method: String, receipt: String?) {
        viewModelScope.launch {
            _collectState.value = UiState.Loading
            _collectState.value = repo.collectPayment(accountId, amount, method, receipt)
        }
    }

    fun resetCollect() {
        _collectState.value = UiState.Idle
        _dueState.value     = UiState.Idle
    }
}
