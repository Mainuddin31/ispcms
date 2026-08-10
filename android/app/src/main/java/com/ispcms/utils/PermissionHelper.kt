package com.ispcms.utils

import com.ispcms.data.models.Permission

/**
 * Evaluates module:action permission pairs against the user's permission list.
 * All business rules are enforced by the backend; this class only drives UI
 * visibility so users don't see buttons they can't use.
 */
class PermissionHelper(private val permissions: List<Permission>) {

    fun has(module: String, action: String): Boolean =
        permissions.any { it.module == module && it.action == action }

    // Convenience shorthands
    fun canView(module: String)   = has(module, "view")
    fun canCreate(module: String) = has(module, "create")
    fun canUpdate(module: String) = has(module, "update")
    fun canDelete(module: String) = has(module, "delete")

    // Named permission checks used across the app
    val canViewDashboard  get() = canView("dashboard")
    val canViewAccounts   get() = canView("accounts")
    val canViewBilling    get() = canView("billing")
    val canViewExpenses   get() = canView("expenses")
    val canViewReports    get() = canView("reports")
    val canViewRouters    get() = canView("routers")
    val canViewNetwork    get() = canView("network")
    val canViewUsers      get() = canView("users")
    val canViewRoles      get() = canView("roles")
    val canCollect        get() = canCreate("billing")
    val canAddExpense     get() = canCreate("expenses")

    companion object {
        val EMPTY = PermissionHelper(emptyList())
    }
}
