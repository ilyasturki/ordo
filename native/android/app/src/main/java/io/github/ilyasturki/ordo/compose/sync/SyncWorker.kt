package io.github.ilyasturki.ordo.compose.sync

import android.content.Context
import androidx.work.BackoffPolicy
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import io.github.ilyasturki.ordo.compose.OrdoApp
import java.util.concurrent.TimeUnit

// Flushes the queue once the network is there, the app open or not.
class SyncWorker(context: Context, params: WorkerParameters) : CoroutineWorker(context, params) {
    override suspend fun doWork(): Result {
        val engine = (applicationContext as OrdoApp).engine
        engine.awaitBoot()
        val s = engine.state.value
        if (!s.signedIn || !s.waiting) return Result.success()
        return when (engine.sync()) {
            Round.OK -> if (engine.state.value.waiting) Result.retry() else Result.success()
            Round.RETRY -> Result.retry()
            Round.STOP -> Result.success()
        }
    }

    companion object {
        fun enqueue(context: Context) {
            val work = OneTimeWorkRequestBuilder<SyncWorker>()
                .setConstraints(Constraints.Builder().setRequiredNetworkType(NetworkType.CONNECTED).build())
                .setBackoffCriteria(BackoffPolicy.EXPONENTIAL, 30, TimeUnit.SECONDS)
                .build()
            WorkManager.getInstance(context).enqueueUniqueWork("sync", ExistingWorkPolicy.KEEP, work)
        }
    }
}
