package com.ispcms.ui.accounts

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Circle
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.*
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.ispcms.data.models.InternetAccount
import com.ispcms.ui.components.*
import com.ispcms.ui.theme.*
import com.ispcms.utils.UiState

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AccountsScreen(viewModel: AccountsViewModel) {
    val state  by viewModel.state.collectAsStateWithLifecycle()
    val search by viewModel.search.collectAsStateWithLifecycle()

    Scaffold(
        topBar = {
            TopAppBar(title = { Text("Internet Accounts") })
        }
    ) { padding ->
        Column(Modifier.padding(padding).fillMaxSize()) {
            OutlinedTextField(
                value         = search,
                onValueChange = viewModel::onSearch,
                placeholder   = { Text("Search accounts…") },
                leadingIcon   = { Icon(Icons.Default.Search, null) },
                singleLine    = true,
                modifier      = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
                shape         = RoundedCornerShape(12.dp)
            )

            PullToRefreshBox(
                isRefreshing = state is UiState.Loading,
                onRefresh    = { viewModel.load() }
            ) {
                when (val s = state) {
                    is UiState.Loading -> LoadingView()
                    is UiState.Error   -> ErrorView(s.message, onRetry = { viewModel.load() })
                    is UiState.Success -> {
                        if (s.data.isEmpty()) {
                            EmptyView("No accounts found")
                        } else {
                            LazyColumn(
                                contentPadding      = PaddingValues(16.dp),
                                verticalArrangement = Arrangement.spacedBy(8.dp)
                            ) {
                                items(s.data, key = { it.id }) { account ->
                                    AccountRow(account)
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
private fun AccountRow(account: InternetAccount) {
    Card(
        shape     = RoundedCornerShape(12.dp),
        elevation = CardDefaults.cardElevation(2.dp),
        modifier  = Modifier.fillMaxWidth()
    ) {
        Row(Modifier.padding(14.dp), verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp)) {

            // Online indicator
            Icon(
                Icons.Default.Circle,
                contentDescription = null,
                tint     = if (account.isOnline) Emerald500 else Slate400,
                modifier = Modifier.size(10.dp)
            )

            Column(Modifier.weight(1f)) {
                Text(account.username, style = MaterialTheme.typography.titleMedium)
                if (!account.comment.isNullOrBlank()) {
                    Text(account.comment, style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    account.routerName?.let {
                        Chip(it)
                    }
                    account.packageName?.let {
                        Chip(it, color = Blue600)
                    }
                }
            }

            Column(horizontalAlignment = Alignment.End) {
                if (account.monthlyCharge > 0) {
                    Text(fmtTk(account.monthlyCharge), style = MaterialTheme.typography.bodyMedium)
                }
                if (account.isOnline && !account.currentIp.isNullOrBlank()) {
                    Text(account.currentIp, style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
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
