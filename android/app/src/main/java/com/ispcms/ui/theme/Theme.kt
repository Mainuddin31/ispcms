package com.ispcms.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable

private val LightColorScheme = lightColorScheme(
    primary          = Blue600,
    onPrimary        = White,
    primaryContainer = Blue100,
    onPrimaryContainer = Navy900,
    secondary        = Emerald500,
    onSecondary      = White,
    background       = Background,
    onBackground     = Navy900,
    surface          = White,
    onSurface        = Navy900,
    surfaceVariant   = Slate100,
    onSurfaceVariant = Navy700,
    error            = Red500,
    onError          = White,
    errorContainer   = Red100,
)

private val DarkColorScheme = darkColorScheme(
    primary          = Blue500,
    onPrimary        = Navy900,
    primaryContainer = Navy700,
    onPrimaryContainer = Blue100,
    secondary        = Emerald500,
    onSecondary      = Navy900,
    background       = Navy900,
    onBackground     = White,
    surface          = Navy800,
    onSurface        = White,
    surfaceVariant   = Navy700,
    onSurfaceVariant = Slate400,
    error            = Red500,
    onError          = White,
    errorContainer   = Color(0xFF7F1D1D),
)

@Composable
fun IspcmsTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit
) {
    val colorScheme = if (darkTheme) DarkColorScheme else LightColorScheme
    MaterialTheme(
        colorScheme = colorScheme,
        typography  = Typography,
        content     = content
    )
}
