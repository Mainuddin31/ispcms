package com.ispcms.ui.main

import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.ispcms.data.repository.AuthRepository
import com.ispcms.ui.accounts.AccountsScreen
import com.ispcms.ui.accounts.AccountsViewModel
import com.ispcms.ui.bills.BillsScreen
import com.ispcms.ui.bills.BillsViewModel
import com.ispcms.ui.dashboard.DashboardScreen
import com.ispcms.ui.dashboard.DashboardViewModel
import com.ispcms.ui.expenses.ExpensesScreen
import com.ispcms.ui.expenses.ExpensesViewModel
import com.ispcms.ui.profile.ProfileScreen
import com.ispcms.ui.reports.ReportScreen
import com.ispcms.ui.reports.ReportViewModel
import com.ispcms.utils.PermissionHelper
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.launch
import javax.inject.Inject

private data class NavTab(
    val label:   String,
    val icon:    ImageVector,
    val require: (PermissionHelper) -> Boolean = { true }
)

private val ALL_TABS = listOf(
    NavTab("Dashboard", Icons.Default.Dashboard),
    NavTab("Accounts",  Icons.Default.People)            { it.canViewAccounts },
    NavTab("Bills",     Icons.Default.Receipt)           { it.canViewBilling },
    NavTab("Report",    Icons.Default.BarChart)          { it.canViewReports },
    NavTab("Expenses",  Icons.Default.MonetizationOn)   { it.canViewExpenses },
    NavTab("Profile",   Icons.Default.Person)
)

@HiltViewModel
class LogoutViewModel @Inject constructor(
    private val authRepo: AuthRepository
) : ViewModel() {
    fun logout(onDone: () -> Unit) {
        viewModelScope.launch {
            authRepo.logout()
            onDone()
        }
    }
}

@Composable
fun MainScreen(onLogout: () -> Unit) {
    val mainVm:   MainViewModel   = hiltViewModel()
    val logoutVm: LogoutViewModel = hiltViewModel()

    // Hoist ALL ViewModels unconditionally — Compose requires composable
    // calls to happen in the same order on every recomposition.
    val dashboardVm: DashboardViewModel = hiltViewModel()
    val accountsVm:  AccountsViewModel  = hiltViewModel()
    val billsVm:     BillsViewModel     = hiltViewModel()
    val reportVm:    ReportViewModel    = hiltViewModel()
    val expensesVm:  ExpensesViewModel  = hiltViewModel()

    val perms = mainVm.permissions
    val user  = mainVm.currentUser

    val visibleTabs = ALL_TABS.filter { it.require(perms) }
    var selectedIdx by rememberSaveable { mutableIntStateOf(0) }
    if (selectedIdx >= visibleTabs.size) selectedIdx = 0

    Scaffold(
        bottomBar = {
            NavigationBar {
                visibleTabs.forEachIndexed { idx, tab ->
                    NavigationBarItem(
                        selected = selectedIdx == idx,
                        onClick  = { selectedIdx = idx },
                        icon     = { Icon(tab.icon, contentDescription = tab.label) },
                        label    = { Text(tab.label) }
                    )
                }
            }
        }
    ) { padding ->
        val mod = Modifier.padding(padding)
        when (visibleTabs.getOrNull(selectedIdx)?.label) {
            "Dashboard" -> DashboardScreen(dashboardVm, perms, user?.name ?: user?.username ?: "")
            "Accounts"  -> AccountsScreen(accountsVm)
            "Bills"     -> BillsScreen(billsVm, perms)
            "Report"    -> ReportScreen(reportVm)
            "Expenses"  -> ExpensesScreen(expensesVm, perms)
            "Profile"   -> ProfileScreen(
                user     = user,
                onLogout = { logoutVm.logout { onLogout() } }
            )
        }
    }
}
