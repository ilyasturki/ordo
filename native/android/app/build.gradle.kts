plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
}

val repo: File = rootProject.projectDir.resolve("../..").canonicalFile

android {
    namespace = "io.github.ilyasturki.ordo.compose"
    compileSdk = 37

    defaultConfig {
        applicationId = "io.github.ilyasturki.ordo.compose"
        minSdk = 26
        targetSdk = 37
        versionCode = 1
        versionName = repo.resolve("VERSION").readText().trim()
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    buildTypes {
        debug {
            applicationIdSuffix = ".dev"
        }
        release {
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

    sourceSets.getByName("androidTest").assets.srcDir(repo.resolve("testdata/web"))
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
