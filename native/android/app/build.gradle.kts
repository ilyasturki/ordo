plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
}

val repo: File = rootProject.projectDir.resolve("../..").canonicalFile

// Set by the release job, so a local assembleRelease leaves the APK unsigned: an APK signed by another key can't update an installed one.
val keystore: String? = System.getenv("ORDO_KEYSTORE")

android {
    namespace = "io.github.ilyasturki.ordo.compose"
    compileSdk = 37

    defaultConfig {
        applicationId = "io.github.ilyasturki.ordo"
        minSdk = 26
        targetSdk = 37
        versionCode = providers.gradleProperty("ordoVersionCode").orNull?.toInt() ?: 1
        versionName = repo.resolve("VERSION").readText().trim()
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    signingConfigs {
        if (keystore != null) {
            create("release") {
                storeFile = file(keystore)
                storePassword = System.getenv("ORDO_KEYSTORE_PASSWORD")
                keyAlias = System.getenv("ORDO_KEY_ALIAS")
                keyPassword = System.getenv("ORDO_KEY_PASSWORD")
            }
        }
    }

    buildTypes {
        debug {
            applicationIdSuffix = ".dev"
        }
        release {
            if (keystore != null) signingConfig = signingConfigs.getByName("release")
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
        }
        // Optimized like a release, under the debug build's id and key: what the phone runs, since a debug build stutters.
        create("phone") {
            initWith(getByName("release"))
            applicationIdSuffix = ".dev"
            signingConfig = signingConfigs.getByName("debug")
            matchingFallbacks += "release"
        }
    }

    buildFeatures {
        compose = true
    }

    testOptions {
        unitTests.all {
            it.systemProperty("ordo.repo", repo.path)
        }
    }

    sourceSets.getByName("androidTest").assets.directories.add(repo.resolve("testdata/native").path)
    // The phone build is a .dev build too: its label and adb sign-in, which a release never has.
    sourceSets.getByName("phone") {
        manifest.srcFile("src/debug/AndroidManifest.xml")
        kotlin.directories.add("src/debug/java")
        res.directories.add("src/debug/res")
    }
}

dependencies {
    implementation(files("libs/ordocore.aar"))
    implementation(platform(libs.compose.bom))
    implementation(libs.compose.ui)
    implementation(libs.compose.foundation)
    implementation(libs.material3)
    implementation(libs.activity.compose)
    implementation(libs.core.ktx)
    implementation(libs.lifecycle.runtime.compose)
    implementation(libs.lifecycle.process)
    implementation(libs.work.runtime)
    implementation(libs.okhttp)
    implementation(libs.okhttp.sse)
    implementation(libs.serialization.json)
    implementation(libs.coroutines.android)
    implementation(libs.reorderable)
    implementation(libs.code.scanner)

    testImplementation(libs.junit)
    testImplementation(libs.coroutines.test)

    androidTestImplementation(platform(libs.compose.bom))
    androidTestImplementation(libs.compose.ui.test.junit4)
    androidTestImplementation(libs.androidx.test.runner)
    androidTestImplementation(libs.androidx.test.junit)
    debugImplementation(libs.compose.ui.test.manifest)
}
