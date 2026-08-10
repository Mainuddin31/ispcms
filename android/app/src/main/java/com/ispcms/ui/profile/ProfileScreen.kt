package com.ispcms.ui.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Logout
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.ispcms.BuildConfig
import com.ispcms.data.models.User
import com.ispcms.ui.theme.*

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ProfileScreen(user: User?, onLogout: () -> Unit) {
    var showLogoutDialog by remember { mutableStateOf(false) }

    if (showLogoutDialog) {
        AlertDialog(
            onDismissRequest = { showLogoutDialog = false },
            title   = { Text("Sign out") },
            text    = { Text("Are you sure you want to sign out?") },
            confirmButton = {
                Button(
                    onClick = onLogout,
                    colors  = ButtonDefaults.buttonColors(containerColor = Red500)
                ) { Text("Sign out") }
            },
            dismissButton = {
                TextButton(onClick = { showLogoutDialog = false }) { Text("Cancel") }
            }
        )
    }

    Scaffold(topBar = { TopAppBar(title = { Text("Profile") }) }) { padding ->
        Column(
            modifier            = Modifier.padding(padding).fillMaxSize().padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(24.dp)
        ) {
            // Avatar
            Box(
                modifier = Modifier
                    .size(96.dp)
                    .clip(CircleShape)
                    .background(Blue600),
                contentAlignment = Alignment.Center
            ) {
                Icon(Icons.Default.Person, null, tint = White, modifier = Modifier.size(52.dp))
            }

            // User info
            Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text(user?.name ?: user?.username ?: "—",
                    style = MaterialTheme.typography.headlineMedium.copy(fontWeight = FontWeight.Bold))
                user?.email?.let {
                    Text(it, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                user?.role?.let {
                    Surface(color = Blue600.copy(alpha = 0.12f), shape = RoundedCornerShape(8.dp)) {
                        Text(it.name.replace("_", " ").replaceFirstChar { c -> c.uppercase() },
                            modifier = Modifier.padding(horizontal = 12.dp, vertical = 4.dp),
                            style = MaterialTheme.typography.labelMedium, color = Blue600)
                    }
                }
            }

            // Permissions
            user?.role?.permissions?.let { perms ->
                if (perms.isNotEmpty()) {
                    Card(Modifier.fillMaxWidth(), shape = RoundedCornerShape(16.dp)) {
                        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text("Permissions", style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.SemiBold))
                            perms.groupBy { it.module }.forEach { (module, actions) ->
                                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween,
                                    verticalAlignment = Alignment.CenterVertically) {
                                    Text(module.replace("_", " ").replaceFirstChar { it.uppercase() },
                                        style = MaterialTheme.typography.bodySmall.copy(fontWeight = FontWeight.Medium))
                                    Text(actions.map { it.action }.joinToString(", "),
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant)
                                }
                            }
                        }
                    }
                }
            }

            Spacer(Modifier.weight(1f))

            // Build info
            Text(
                "Build: ${BuildConfig.FLAVOR} · v${BuildConfig.VERSION_NAME}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )

            // Logout
            Button(
                onClick  = { showLogoutDialog = true },
                modifier = Modifier.fillMaxWidth().height(52.dp),
                shape    = RoundedCornerShape(12.dp),
                colors   = ButtonDefaults.buttonColors(containerColor = Red500)
            ) {
                Icon(Icons.Default.Logout, null, modifier = Modifier.size(18.dp))
                Spacer(Modifier.width(8.dp))
                Text("Sign Out", style = MaterialTheme.typography.titleMedium)
            }
        }
    }
}
