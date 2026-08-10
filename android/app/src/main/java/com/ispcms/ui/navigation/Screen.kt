package com.ispcms.ui.navigation

sealed class Screen(val route: String) {
    data object Login          : Screen("login")
    data object Dashboard      : Screen("dashboard")
    data object Accounts       : Screen("accounts")
    data object Bills          : Screen("bills")
    data object CollectionReport : Screen("collection_report")
    data object Expenses       : Screen("expenses")
    data object Profile        : Screen("profile")
}
