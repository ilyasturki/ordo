package io.github.ilyasturki.ordo.compose.ui

import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.hapticfeedback.HapticFeedback
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.platform.LocalHapticFeedback

// kilorep's feedback (src/lib/ui/feedback.ts) on the platform's haptics, so the system's touch
// feedback setting governs it; taps get the system click from Compose's clickable.
class Feedback(private val haptics: HapticFeedback) {
    fun commit() = haptics.performHapticFeedback(HapticFeedbackType.Confirm)
    fun hold() = haptics.performHapticFeedback(HapticFeedbackType.LongPress)
    fun tick() = haptics.performHapticFeedback(HapticFeedbackType.SegmentTick)
    fun slot() = haptics.performHapticFeedback(HapticFeedbackType.SegmentFrequentTick)
    fun reject() = haptics.performHapticFeedback(HapticFeedbackType.Reject)
}

@Composable
fun rememberFeedback(): Feedback {
    val haptics = LocalHapticFeedback.current
    return remember(haptics) { Feedback(haptics) }
}
