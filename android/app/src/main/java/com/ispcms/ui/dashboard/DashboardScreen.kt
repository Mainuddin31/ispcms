package com.ispcms.ui.dashboard

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.ispcms.data.models.DashboardStats
import com.ispcms.ui.components.*
import com.ispcms.ui.theme.*
import com.ispcms.utils.PermissionHelper
import com.ispcms.utils.UiState

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(
    viewModel:   DashboardViewModel,
    permissions: PermissionHelper,
    userName:    String
) {
    val statsState by viewModel.stats.collectAsStateWithLifecycle()
    val isRefreshing = statsState is UiState.Loading

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text("Dashboard", style = MaterialTheme.typography.titleLarge)
                        Text("Welcome, $userName", style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            )
        }
    ) { padding ->
        PullToRefreshBox(
            isRefreshing = isRefreshing,
            onRefresh    = { viewModel.load() },
            modifier     = Modifier.padding(padding)
        ) {
            when (val s = statsState) {
                is UiState.Loading -> LoadingView()
                is UiState.Error   -> ErrorView(s.message, onRetry = { viewModel.load() })
                is UiState.Success -> DashboardContent(s.data, permissions)
                else               -> {}
            }
        }
    }
}

@Composable
private fun DashboardContent(stats: DashboardStats, perms: PermissionHelper) {
    LazyColumn(
        modifier            = Modifier.fillMaxSize(),
        contentPadding      = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        // ── Collections ─────────────────────────────────────────────────────
        if (perms.canViewBilling) {
            item {
                SectionHeader("Collections")
                Spacer(Modifier.height(4.dp))
                Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        StatCard(
                            title     = "Today's Collection",
                            value     = fmtTk(stats.todayCollection),
                            icon      = Icons.Default.ArrowDownward,
                            iconColor = Emerald500,
                            modifier  = Modifier.weight(1f)
                        )
                        StatCard(
                            title     = "Outstanding Due",
                            value     = fmtTk(stats.totalOutstandingDue),
                            icon      = Icons.Default.Warning,
                            iconColor = if (stats.totalOutstandingDue > 0) Orange500 else Emerald500,
                            modifier  = Modifier.weight(1f)
                        )
                    }
                    StatCard(
                        title     = "Total Collected",
                        value     = fmtTk(stats.monthlyCollection),
                        icon      = Icons.Default.TrendingUp,
                        iconColor = Green600,
                        sub       = "Last month ${fmtTk(stats.lastMonthCollection)}",
                        modifier  = Modifier.fillMaxWidth()
                    )
                }
            }
        }

        // ── Expenses ────────────────────────────────────────────────────────
        if (perms.canViewExpenses) {
            item {
                SectionHeader("Expenses")
                Spacer(Modifier.height(4.dp))
                Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        StatCard(
                            title     = "Today's Expense",
                            value     = fmtTk(stats.todayExpense),
                            icon      = Icons.Default.ArrowUpward,
                            iconColor = Orange500,
                            modifier  = Modifier.weight(1f)
                        )
                        StatCard(
                            title     = "Cash in Hand",
                            value     = fmtTk(stats.monthlyCollection - stats.monthlyExpense),
                            icon      = Icons.Default.AccountBalance,
                            iconColor = Blue600,
                            modifier  = Modifier.weight(1f)
                        )
                    }
                    StatCard(
                        title     = "Total Expense",
                        value     = fmtTk(stats.monthlyExpense),
                        icon      = Icons.Default.Receipt,
                        iconColor = Amber500,
                        sub       = "Last month ${fmtTk(stats.lastMonthExpense)}",
                        modifier  = Modifier.fillMaxWidth()
                    )
                }
            }
        }

        // Cash in Hand for billing officers without expenses perm
        if (perms.canViewBilling && !perms.canViewExpenses) {
            item {
                StatCard(
                    title     = "Cash in Hand",
                    value     = fmtTk(stats.monthlyCollection - stats.monthlyExpense),
                    icon      = Icons.Default.AccountBalance,
                    iconColor = Blue600,
                    sub       = "This month collection − expense",
                    modifier  = Modifier.fillMaxWidth()
                )
            }
        }

        // ── Billing stats ───────────────────────────────────────────────────
        if (perms.canViewBilling) {
            item {
                SectionHeader("Billing Overview")
                Spacer(Modifier.height(4.dp))
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    CountCard("Bills", stats.totalBills, modifier = Modifier.weight(1f))
                    CountCard("Paid", stats.paidBills, color = Emerald500, modifier = Modifier.weight(1f))
                    CountCard("Unpaid", stats.unpaidBills, color = Red500, modifier = Modifier.weight(1f))
                }
            }
        }

        // ── Network / Routers ────────────────────────────────────────────────
        if (perms.canViewRouters) {
            item {
                SectionHeader("Network")
                Spacer(Modifier.height(4.dp))
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    StatCard(
                        title     = "Routers",
                        value     = "${stats.onlineRouters}/${stats.totalRouters}",
                        icon      = Icons.Default.Router,
                        iconColor = if (stats.onlineRouters == stats.totalRouters) Emerald500 else Orange500,
                        sub       = "Online / Total",
                        modifier  = Modifier.weight(1f)
                    )
                    if (perms.canViewAccounts) {
                        StatCard(
                            title     = "Accounts",
                            value     = "${stats.onlineAccounts}/${stats.totalAccounts}",
                            icon      = Icons.Default.People,
                            iconColor = Blue500,
                            sub       = "Online / Total",
                            modifier  = Modifier.weight(1f)
                        )
                    }
                }
            }
        }

        // ── Recent Activity ──────────────────────────────────────────────────
        if (stats.recentActivity.isNotEmpty()) {
            item { SectionHeader("Recent Activity") }
            items(stats.recentActivity.take(5)) { item ->
                ActivityRow(actor = item.actorName ?: "System", description = item.description, time = item.createdAt)
            }
        }

        // ── Recent Syncs ─────────────────────────────────────────────────────
        if (perms.canViewRouters && stats.recentSyncs.isNotEmpty()) {
            item { SectionHeader("Recent Syncs") }
            items(stats.recentSyncs.take(5)) { sync ->
                SyncRow(routerName = sync.routerName, status = sync.status, time = sync.syncedAt)
            }
        }

        item { Spacer(Modifier.height(16.dp)) }
    }
}

@Composable
private fun CountCard(label: String, count: Int, color: Color = Blue600, modifier: Modifier = Modifier) {
    Card(modifier = modifier, shape = RoundedCornerShape(12.dp), elevation = CardDefaults.cardElevation(2.dp)) {
        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(count.toString(), style = MaterialTheme.typography.titleLarge.copy(fontWeight = FontWeight.Bold), color = color)
            Text(label, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@Composable
private fun ActivityRow(actor: String, description: String, time: String) {
    Row(Modifier.fillMaxWidth().padding(vertical = 4.dp), horizontalArrangement = Arrangement.SpaceBetween) {
        Column(Modifier.weight(1f)) {
            Text(actor, style = MaterialTheme.typography.bodySmall.copy(fontWeight = FontWeight.Medium))
            Text(description, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        Text(time.take(10), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

@Composable
private fun SyncRow(routerName: String, status: String, time: String) {
    Row(Modifier.fillMaxWidth().padding(vertical = 4.dp), horizontalArrangement = Arrangement.SpaceBetween) {
        Text(routerName, style = MaterialTheme.typography.bodySmall.copy(fontWeight = FontWeight.Medium))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            val color = if (status == "success") Emerald500 else Red500
            Surface(color = color.copy(alpha = 0.15f), shape = RoundedCornerShape(4.dp)) {
                Text(status, modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
                    style = MaterialTheme.typography.labelMedium, color = color)
            }
            Text(time.take(10), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}
