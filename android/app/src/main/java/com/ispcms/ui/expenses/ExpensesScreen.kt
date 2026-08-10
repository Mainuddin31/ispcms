package com.ispcms.ui.expenses

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material3.*
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.ispcms.data.models.Expense
import com.ispcms.ui.components.*
import com.ispcms.ui.theme.*
import com.ispcms.utils.PermissionHelper
import com.ispcms.utils.UiState
import java.time.LocalDate
import java.time.format.DateTimeFormatter

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ExpensesScreen(viewModel: ExpensesViewModel, permissions: PermissionHelper) {
    val state      by viewModel.expenses.collectAsStateWithLifecycle()
    val categories by viewModel.categories.collectAsStateWithLifecycle()
    val addState   by viewModel.addState.collectAsStateWithLifecycle()
    var showAdd    by remember { mutableStateOf(false) }

    LaunchedEffect(addState) {
        if (addState is UiState.Success) showAdd = false
    }

    if (showAdd) {
        AddExpenseDialog(
            categories = categories,
            addState   = addState,
            onAdd      = { date, catId, amount, desc, vendor, method, ref ->
                viewModel.addExpense(date, catId, amount, desc, vendor, method, ref)
            },
            onDismiss  = { showAdd = false; viewModel.resetAddState() }
        )
    }

    Scaffold(
        topBar = { TopAppBar(title = { Text("Expenses") }) },
        floatingActionButton = {
            if (permissions.canAddExpense) {
                FloatingActionButton(onClick = { showAdd = true }, containerColor = Blue600) {
                    Icon(Icons.Default.Add, null, tint = White)
                }
            }
        }
    ) { padding ->
        PullToRefreshBox(isRefreshing = state is UiState.Loading, onRefresh = { viewModel.load() },
            modifier = Modifier.padding(padding)) {
            when (val s = state) {
                is UiState.Loading -> LoadingView()
                is UiState.Error   -> ErrorView(s.message, onRetry = { viewModel.load() })
                is UiState.Success -> {
                    if (s.data.isEmpty()) EmptyView("No expenses recorded")
                    else LazyColumn(contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        items(s.data, key = { it.id }) { expense -> ExpenseRow(expense) }
                    }
                }
                else -> {}
            }
        }
    }
}

@Composable
private fun ExpenseRow(expense: Expense) {
    Card(Modifier.fillMaxWidth(), shape = RoundedCornerShape(12.dp), elevation = CardDefaults.cardElevation(2.dp)) {
        Row(Modifier.padding(14.dp), verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween) {
            Column(Modifier.weight(1f)) {
                Text(expense.description ?: expense.category ?: "Expense",
                    style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.SemiBold))
                expense.vendor?.let {
                    Text(it, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    expense.category?.let { Chip(it) }
                    expense.paymentMethod?.let { Chip(it, Blue600) }
                }
            }
            Column(horizontalAlignment = Alignment.End) {
                Text(fmtTk(expense.amount), style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                    color = Orange500)
                Text(expense.date.take(10), style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}

@Composable
private fun Chip(label: String, color: androidx.compose.ui.graphics.Color = Slate400) {
    Surface(color = color.copy(alpha = 0.12f), shape = RoundedCornerShape(4.dp)) {
        Text(label, modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
            style = MaterialTheme.typography.labelMedium, color = color)
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun AddExpenseDialog(
    categories: List<com.ispcms.data.models.ExpenseCategory>,
    addState:   UiState<*>,
    onAdd:      (String, String?, Double, String?, String?, String, String?) -> Unit,
    onDismiss:  () -> Unit
) {
    val today = LocalDate.now().format(DateTimeFormatter.ISO_LOCAL_DATE)
    var date        by remember { mutableStateOf(today) }
    var amount      by remember { mutableStateOf("") }
    var description by remember { mutableStateOf("") }
    var vendor      by remember { mutableStateOf("") }
    var method      by remember { mutableStateOf("cash") }
    var reference   by remember { mutableStateOf("") }
    var selectedCat by remember { mutableStateOf<String?>(null) }
    var expanded    by remember { mutableStateOf(false) }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Add Expense") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                OutlinedTextField(value = date, onValueChange = { date = it },
                    label = { Text("Date (YYYY-MM-DD)") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(value = amount, onValueChange = { amount = it }, label = { Text("Amount (৳)") },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                    singleLine = true, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(value = description, onValueChange = { description = it },
                    label = { Text("Description") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(value = vendor, onValueChange = { vendor = it },
                    label = { Text("Vendor (optional)") }, singleLine = true, modifier = Modifier.fillMaxWidth())

                if (categories.isNotEmpty()) {
                    ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = !expanded }) {
                        OutlinedTextField(
                            readOnly = true,
                            value = categories.find { it.id == selectedCat }?.name ?: "Select Category",
                            onValueChange = {},
                            label = { Text("Category") },
                            trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded) },
                            modifier = Modifier.menuAnchor().fillMaxWidth()
                        )
                        ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                            DropdownMenuItem(text = { Text("None") }, onClick = { selectedCat = null; expanded = false })
                            categories.forEach { cat ->
                                DropdownMenuItem(text = { Text(cat.name) }, onClick = { selectedCat = cat.id; expanded = false })
                            }
                        }
                    }
                }

                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    listOf("cash", "bkash", "bank").forEach { m ->
                        FilterChip(selected = method == m, onClick = { method = m },
                            label = { Text(m.replaceFirstChar { it.uppercase() }) })
                    }
                }

                OutlinedTextField(value = reference, onValueChange = { reference = it },
                    label = { Text("Reference # (optional)") }, singleLine = true, modifier = Modifier.fillMaxWidth())

                if (addState is UiState.Error) {
                    Text(addState.message, color = Red500, style = MaterialTheme.typography.bodySmall)
                }
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    val amt = amount.toDoubleOrNull() ?: return@Button
                    onAdd(date, selectedCat, amt, description.takeIf { it.isNotBlank() },
                        vendor.takeIf { it.isNotBlank() }, method, reference.takeIf { it.isNotBlank() })
                },
                enabled = addState !is UiState.Loading,
                colors  = ButtonDefaults.buttonColors(containerColor = Blue600)
            ) {
                if (addState is UiState.Loading) CircularProgressIndicator(Modifier.size(16.dp), White, 2.dp)
                else Text("Add")
            }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } }
    )
}
