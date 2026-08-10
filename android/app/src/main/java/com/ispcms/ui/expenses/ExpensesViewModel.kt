package com.ispcms.ui.expenses

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.ispcms.data.models.CreateExpenseRequest
import com.ispcms.data.models.Expense
import com.ispcms.data.models.ExpenseCategory
import com.ispcms.data.repository.ExpenseRepository
import com.ispcms.utils.UiState
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class ExpensesViewModel @Inject constructor(
    private val repo: ExpenseRepository
) : ViewModel() {

    private val _expenses   = MutableStateFlow<UiState<List<Expense>>>(UiState.Loading)
    val expenses: StateFlow<UiState<List<Expense>>> = _expenses.asStateFlow()

    private val _categories = MutableStateFlow<List<ExpenseCategory>>(emptyList())
    val categories: StateFlow<List<ExpenseCategory>> = _categories.asStateFlow()

    private val _addState   = MutableStateFlow<UiState<Expense>>(UiState.Idle)
    val addState: StateFlow<UiState<Expense>> = _addState.asStateFlow()

    init {
        load()
        loadCategories()
    }

    fun load() {
        viewModelScope.launch {
            _expenses.value = UiState.Loading
            val res = repo.getExpenses()
            _expenses.value = when (res) {
                is UiState.Success -> UiState.Success(res.data.data)
                is UiState.Error   -> UiState.Error(res.message)
                else               -> UiState.Error("Unexpected")
            }
        }
    }

    private fun loadCategories() {
        viewModelScope.launch {
            val res = repo.getCategories()
            if (res is UiState.Success) _categories.value = res.data
        }
    }

    fun addExpense(date: String, categoryId: String?, amount: Double, description: String?,
                   vendor: String?, method: String, reference: String?) {
        viewModelScope.launch {
            _addState.value = UiState.Loading
            _addState.value = repo.createExpense(
                CreateExpenseRequest(date, categoryId, amount, description, vendor, method, reference)
            )
            if (_addState.value is UiState.Success) load()
        }
    }

    fun resetAddState() { _addState.value = UiState.Idle }
}
