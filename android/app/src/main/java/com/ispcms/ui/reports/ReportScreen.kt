package com.ispcms.ui.reports

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.ChevronLeft
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material3.*
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.ispcms.data.models.CollectionReportResponse
import com.ispcms.data.models.CollectionReportRow
import com.ispcms.ui.components.*
import com.ispcms.ui.theme.*
import com.ispcms.utils.UiState
import java.util.Calendar

private val MONTHS = listOf("Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReportScreen(viewModel: ReportViewModel) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val month by viewModel.month.collectAsStateWithLifecycle()
    val year  by viewModel.year.collectAsStateWithLifecycle()

    Scaffold(topBar = { TopAppBar(title = { Text("Collection Report") }) }) { padding ->
        Column(Modifier.padding(padding).fillMaxSize()) {
            // Month navigator
            MonthNavigator(month, year, onPrev = {
                if (month == 1) viewModel.setMonth(12, year - 1)
                else viewModel.setMonth(month - 1, year)
            }, onNext = {
                val now = Calendar.getInstance()
                val currentMonth = now.get(Calendar.MONTH) + 1
                val currentYear  = now.get(Calendar.YEAR)
                if (year < currentYear || (year == currentYear && month < currentMonth)) {
                    if (month == 12) viewModel.setMonth(1, year + 1)
                    else viewModel.setMonth(month + 1, year)
                }
            })

            PullToRefreshBox(isRefreshing = state is UiState.Loading, onRefresh = { viewModel.load() }) {
                when (val s = state) {
                    is UiState.Loading -> LoadingView()
                    is UiState.Error   -> ErrorView(s.message, onRetry = { viewModel.load() })
                    is UiState.Success -> ReportContent(s.data)
                    else -> {}
                }
            }
        }
    }
}

@Composable
private fun MonthNavigator(month: Int, year: Int, onPrev: () -> Unit, onNext: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        IconButton(onClick = onPrev) { Icon(Icons.Default.ChevronLeft, null) }
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            Icon(Icons.Default.CalendarMonth, null, tint = Blue600)
            Text("${MONTHS[month - 1]} $year", style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.SemiBold))
        }
        IconButton(onClick = onNext) { Icon(Icons.Default.ChevronRight, null) }
    }
}

@Composable
private fun ReportContent(report: CollectionReportResponse) {
    val summary = report.summary
    LazyColumn(
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        // Summary cards
        item {
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                StatCard("Total Active", summary.totalActive.toString(), Icons.Default.People, Blue600, Modifier.weight(1f))
                StatCard("Collected", summary.totalCollected.toString(), Icons.Default.CheckCircle, Emerald500, Modifier.weight(1f))
            }
        }
        item {
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                StatCard("Collection", fmtTk(summary.collectionAmount), Icons.Default.Paid, Green600, Modifier.weight(1f))
                StatCard("Total Due", fmtTk(summary.totalDue), Icons.Default.Warning,
                    if (summary.totalDue > 0) Red500 else Emerald500, Modifier.weight(1f))
            }
        }
        item {
            Card(Modifier.fillMaxWidth(), shape = RoundedCornerShape(12.dp)) {
                Row(Modifier.padding(14.dp), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
                    Text("Collection Rate", style = MaterialTheme.typography.bodyMedium)
                    Text("%.1f%%".format(summary.collectionRate),
                        style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                        color = if (summary.collectionRate >= 80) Emerald500 else Orange500)
                }
                LinearProgressIndicator(
                    progress = { (summary.collectionRate / 100).toFloat().coerceIn(0f, 1f) },
                    modifier = Modifier.fillMaxWidth().padding(horizontal = 14.dp).padding(bottom = 14.dp).height(6.dp),
                    color    = if (summary.collectionRate >= 80) Emerald500 else Orange500,
                    trackColor = MaterialTheme.colorScheme.surfaceVariant
                )
            }
        }

        // Staff cards
        if (report.staffCards.isNotEmpty()) {
            item { SectionHeader("Staff Collection") }
            items(report.staffCards) { staff ->
                Card(Modifier.fillMaxWidth(), shape = RoundedCornerShape(12.dp)) {
                    Row(Modifier.padding(14.dp), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
                        Column {
                            Text(staff.staffName, style = MaterialTheme.typography.titleMedium)
                            Text("${staff.count} clients", style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                        Text(fmtTk(staff.amount), style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                            color = Green600)
                    }
                }
            }
        }

        // Table
        if (report.rows.isNotEmpty()) {
            item { SectionHeader("Details") }
            items(report.rows) { row -> ReportRow(row) }
        }

        item { Spacer(Modifier.height(16.dp)) }
    }
}

@Composable
private fun ReportRow(row: CollectionReportRow) {
    val statusColor = when (row.status) {
        "paid"    -> Emerald500
        "partial" -> Amber500
        "unpaid"  -> Red500
        else      -> Slate400
    }
    Card(Modifier.fillMaxWidth(), shape = RoundedCornerShape(10.dp)) {
        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
                Text(row.username, style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.SemiBold))
                Surface(color = statusColor.copy(alpha = 0.15f), shape = RoundedCornerShape(4.dp)) {
                    Text(row.status.replaceFirstChar { it.uppercase() },
                        modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
                        style = MaterialTheme.typography.labelMedium, color = statusColor)
                }
            }
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Text("Bill: ${fmtTk(row.billAmount)}", style = MaterialTheme.typography.bodySmall)
                Text("Paid: ${fmtTk(row.paidAmount)}", style = MaterialTheme.typography.bodySmall, color = Emerald500)
                Text("Due: ${fmtTk(row.dueAmount)}", style = MaterialTheme.typography.bodySmall,
                    color = if (row.dueAmount > 0) Red500 else Emerald500)
            }
            row.collectedBy?.let {
                Text("Collected by $it", style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}

