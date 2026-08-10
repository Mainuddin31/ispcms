package com.ispcms.ui.bills

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Payment
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.*
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.ispcms.data.models.AccountDueResponse
import com.ispcms.data.models.Bill
import com.ispcms.ui.components.*
import com.ispcms.ui.theme.*
import com.ispcms.utils.PermissionHelper
import com.ispcms.utils.UiState

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BillsScreen(viewModel: BillsViewModel, permissions: PermissionHelper) {
    val state        by viewModel.bills.collectAsStateWithLifecycle()
    val search       by viewModel.search.collectAsStateWithLifecycle()
    val dueState     by viewModel.dueState.collectAsStateWithLifecycle()
    val collectState by viewModel.collectState.collectAsStateWithLifecycle()

    var collectAccountId by remember { mutableStateOf<String?>(null) }

    // Reload after successful collect
    LaunchedEffect(collectState) {
        if (collectState is UiState.Success<*>) viewModel.load()
    }

    if (collectAccountId != null) {
        CollectDialog(
            dueState     = dueState,
            collectState = collectState,
            onConfirm    = { amount, method, receipt ->
                viewModel.collect(collectAccountId!!, amount, method, receipt)
            },
            onDismiss = {
                collectAccountId = null
                viewModel.resetCollect()
            }
        )
    }

    Scaffold(topBar = { TopAppBar(title = { Text("Bills") }) }) { padding ->
        Column(Modifier.padding(padding).fillMaxSize()) {
            OutlinedTextField(
                value         = search,
                onValueChange = viewModel::onSearch,
                placeholder   = { Text("Search bills…") },
                leadingIcon   = { Icon(Icons.Default.Search, null) },
                singleLine    = true,
                modifier      = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
                shape         = RoundedCornerShape(12.dp)
            )

            PullToRefreshBox(isRefreshing = state is UiState.Loading, onRefresh = { viewModel.load() }) {
                when (val s = state) {
                    is UiState.Loading    -> LoadingView()
                    is UiState.Error      -> ErrorView(s.message, onRetry = { viewModel.load() })
                    is UiState.Success<*> -> {
                        @Suppress("UNCHECKED_CAST")
                        val bills = s.data as List<Bill>
                        if (bills.isEmpty()) {
                            EmptyView("No bills found")
                        } else {
                            LazyColumn(
                                contentPadding      = PaddingValues(16.dp),
                                verticalArrangement = Arrangement.spacedBy(8.dp)
                            ) {
                                items(bills, key = { it.id }) { bill ->
                                    BillRow(
                                        bill       = bill,
                                        canCollect = permissions.canCollect,
                                        onCollect  = {
                                            collectAccountId = bill.internetAccountId
                                            viewModel.loadDue(bill.internetAccountId)
                                        }
                                    )
                                }
                            }
                        }
                    }
                    else -> {}
                }
            }
        }
    }
}

@Composable
private fun BillRow(bill: Bill, canCollect: Boolean, onCollect: () -> Unit) {
    val statusColor = when (bill.status) {
        "paid"    -> Emerald500
        "partial" -> Amber500
        else      -> Red500
    }

    Card(
        shape     = RoundedCornerShape(12.dp),
        elevation = CardDefaults.cardElevation(2.dp),
        modifier  = Modifier.fillMaxWidth()
    ) {
        Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(
                verticalAlignment       = Alignment.CenterVertically,
                horizontalArrangement   = Arrangement.SpaceBetween,
                modifier                = Modifier.fillMaxWidth()
            ) {
                Column {
                    Text(bill.username, style = MaterialTheme.typography.titleMedium)
                    Text(
                        bill.billMonth,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                Surface(color = statusColor.copy(alpha = 0.15f), shape = RoundedCornerShape(6.dp)) {
                    Text(
                        bill.status.replaceFirstChar { it.uppercase() },
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                        style    = MaterialTheme.typography.labelMedium,
                        color    = statusColor
                    )
                }
            }

            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                AmountColumn("Bill", fmtTk(bill.amount))
                AmountColumn("Paid", fmtTk(bill.paidAmount), Emerald500)
                AmountColumn("Due",  fmtTk(bill.dueAmount),
                    if (bill.dueAmount > 0) Red500 else Emerald500)
            }

            if (canCollect && bill.status != "paid") {
                Button(
                    onClick  = onCollect,
                    modifier = Modifier.fillMaxWidth().height(40.dp),
                    shape    = RoundedCornerShape(8.dp),
                    colors   = ButtonDefaults.buttonColors(containerColor = Blue600)
                ) {
                    Icon(Icons.Default.Payment, null, modifier = Modifier.size(16.dp))
                    Spacer(Modifier.width(6.dp))
                    Text("Collect Payment")
                }
            }
        }
    }
}

@Composable
private fun AmountColumn(
    label: String,
    value: String,
    color: androidx.compose.ui.graphics.Color = MaterialTheme.colorScheme.onSurface
) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Text(label, style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(value, style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.SemiBold),
            color = color)
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun CollectDialog(
    dueState:     UiState<AccountDueResponse>,
    collectState: UiState<*>,
    onConfirm:    (Double, String, String?) -> Unit,
    onDismiss:    () -> Unit
) {
    var amount  by remember { mutableStateOf("") }
    var method  by remember { mutableStateOf("cash") }
    var receipt by remember { mutableStateOf("") }

    // Pre-fill total outstanding when due bills load
    LaunchedEffect(dueState) {
        if (dueState is UiState.Success) {
            amount = dueState.data.totalOutstanding.toLong().toString()
        }
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Collect Payment") },
        text  = {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {

                when (dueState) {
                    is UiState.Loading -> LinearProgressIndicator(Modifier.fillMaxWidth())
                    is UiState.Success -> {
                        dueState.data.bills.forEach { bill ->
                            Row(
                                Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween
                            ) {
                                Text(bill.billMonth, style = MaterialTheme.typography.bodySmall)
                                Text(
                                    "Due: ${fmtTk(bill.dueAmount)}",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = if (bill.dueAmount > 0) Red500 else Emerald500
                                )
                            }
                        }
                        HorizontalDivider()
                    }
                    else -> {}
                }

                OutlinedTextField(
                    value           = amount,
                    onValueChange   = { amount = it },
                    label           = { Text("Amount (৳)") },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                    singleLine      = true,
                    modifier        = Modifier.fillMaxWidth()
                )

                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    listOf("Cash", "bKash", "Bank").forEachIndexed { i, label ->
                        val value = listOf("cash", "bkash", "bank")[i]
                        FilterChip(
                            selected = method == value,
                            onClick  = { method = value },
                            label    = { Text(label) }
                        )
                    }
                }

                OutlinedTextField(
                    value         = receipt,
                    onValueChange = { receipt = it },
                    label         = { Text("Receipt # (optional)") },
                    singleLine    = true,
                    modifier      = Modifier.fillMaxWidth()
                )

                if (collectState is UiState.Error) {
                    Text(collectState.message, color = Red500,
                        style = MaterialTheme.typography.bodySmall)
                }
                if (collectState is UiState.Success<*>) {
                    Text("Payment collected successfully!", color = Emerald500,
                        style = MaterialTheme.typography.bodySmall)
                }
            }
        },
        confirmButton = {
            Button(
                onClick  = {
                    val amt = amount.toDoubleOrNull() ?: return@Button
                    onConfirm(amt, method, receipt.takeIf { it.isNotBlank() })
                },
                enabled = collectState !is UiState.Loading,
                colors  = ButtonDefaults.buttonColors(containerColor = Blue600)
            ) {
                if (collectState is UiState.Loading) {
                    CircularProgressIndicator(
                        modifier    = Modifier.size(16.dp),
                        color       = White,
                        strokeWidth = 2.dp
                    )
                } else {
                    Text("Collect")
                }
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text("Cancel") }
        }
    )
}
