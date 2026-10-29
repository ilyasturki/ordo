package io.github.ilyasturki.ordo.compose.ui

import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.ColorScheme
import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.MaterialExpressiveTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.MotionScheme
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.runtime.ReadOnlyComposable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontVariation
import androidx.compose.ui.text.font.FontWeight
import io.github.ilyasturki.ordo.compose.R

private fun flex(weight: Int) = Font(
    R.font.google_sans_flex,
    weight = FontWeight(weight),
    variationSettings = FontVariation.Settings(FontVariation.weight(weight)),
)

val GoogleSansFlex = FontFamily(flex(400), flex(500), flex(600), flex(700))

private fun TextStyle.flex() = copy(fontFamily = GoogleSansFlex)

private val typography = Typography().run {
    copy(
        displayLarge = displayLarge.flex(),
        displayMedium = displayMedium.flex(),
        displaySmall = displaySmall.flex(),
        headlineLarge = headlineLarge.flex(),
        headlineMedium = headlineMedium.flex(),
        headlineSmall = headlineSmall.flex(),
        titleLarge = titleLarge.flex(),
        titleMedium = titleMedium.flex(),
        titleSmall = titleSmall.flex(),
        bodyLarge = bodyLarge.flex(),
        bodyMedium = bodyMedium.flex(),
        bodySmall = bodySmall.flex(),
        labelLarge = labelLarge.flex(),
        labelMedium = labelMedium.flex(),
        labelSmall = labelSmall.flex(),
    )
}

// The four tag dots and the priority marks, the web app's tokens (app/src/app.css).
data class Marks(val green: Color, val blue: Color, val magenta: Color, val cyan: Color, val up: Color, val down: Color) {
    fun tag(color: String): Color = when (color.removeSuffix("-diamond")) {
        "green" -> green
        "blue" -> blue
        "magenta" -> magenta
        "cyan" -> cyan
        else -> Color.Unspecified
    }
}

private val lightMarks = Marks(Color(0xFF1E7C43), Color(0xFF3557C8), Color(0xFF9A2F8D), Color(0xFF006A6F), Color(0xFFBA1A1A), Color(0xFF006A6F))
private val darkMarks = Marks(Color(0xFF7DD99C), Color(0xFFB3C5FF), Color(0xFFF2A6E4), Color(0xFF6ED5DB), Color(0xFFFFB4AB), Color(0xFF6ED5DB))

private val LocalMarks = staticCompositionLocalOf { lightMarks }

val MaterialTheme.marks: Marks
    @Composable @ReadOnlyComposable get() = LocalMarks.current

@OptIn(ExperimentalMaterial3ExpressiveApi::class)
@Composable
fun OrdoTheme(dark: Boolean = isSystemInDarkTheme(), content: @Composable () -> Unit) {
    val ctx = LocalContext.current
    val scheme: ColorScheme = when {
        Build.VERSION.SDK_INT >= Build.VERSION_CODES.S && dark -> dynamicDarkColorScheme(ctx)
        Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> dynamicLightColorScheme(ctx)
        dark -> darkColorScheme()
        else -> lightColorScheme()
    }
    CompositionLocalProvider(LocalMarks provides if (dark) darkMarks else lightMarks) {
        MaterialExpressiveTheme(colorScheme = scheme, motionScheme = MotionScheme.expressive(), typography = typography, content = content)
    }
}
