# IBMS Android App

Mobile client for the ISP Business Management System. Communicates with the existing Go backend over HTTPS REST. All business logic and authorization live in the backend — the app handles only display and user interaction.

**Stack:** Kotlin · Jetpack Compose · Hilt · Retrofit · OkHttp · EncryptedSharedPreferences · Android Keystore

---

## Screens

| Screen | Permission Required |
|--------|-------------------|
| Login | — |
| Dashboard | `dashboard: view` (always shown) |
| Internet Accounts | `accounts: view` |
| Bills + Collect Payment | `billing: view` |
| Collection Report | `reports: view` |
| Expenses | `expenses: view` |
| Profile + Logout | Always shown |

The bottom navigation shows only tabs the logged-in user has permission to access. Permission data comes from the backend after login — nothing is hardcoded in the app.

---

## First-Time Setup

### 1. Configure API URLs

```bash
cd android/
cp local.properties.example local.properties
nano local.properties
```

Set your actual server addresses:

```properties
# Android emulator → host machine
dev.api.url=http://10.0.2.2:8082/api/v1

# Staging server
staging.api.url=https://staging.example.com/api/v1

# Production server
prod.api.url=https://103.xxx.xxx.xxx/api/v1
```

`local.properties` is gitignored — never commit it.

### 2. Open in Android Studio

File → Open → select the `android/` directory. Android Studio will sync Gradle automatically.

### 3. Download Gradle wrapper (first time only)

```bash
cd android/
gradle wrapper --gradle-version 8.7
```

---

## Build Variants

| Variant | API URL | App ID suffix | App name |
|---------|---------|---------------|---------|
| `developmentDebug` | `dev.api.url` in local.properties | `.dev` | IBMS Dev |
| `stagingRelease` | `staging.api.url` | `.staging` | IBMS Staging |
| `productionRelease` | `prod.api.url` | (none) | IBMS |

The URL is baked into the APK at build time via `BuildConfig.API_BASE_URL`. End users never enter or see the server address.

### Build from Android Studio

- Select variant in the **Build Variants** panel (bottom-left)
- Run → Run 'app' for emulator/device
- Build → Generate Signed Bundle/APK for a distributable APK

### Build from command line

```bash
cd android/

# Development debug APK (emulator)
./gradlew assembleDevelopmentDebug

# Production release APK (requires signing config)
./gradlew assembleProductionRelease
```

APKs output to `app/build/outputs/apk/`.

---

## Signing a Release APK

1. Generate a keystore (one-time):
   ```bash
   keytool -genkey -v -keystore ibms.keystore -alias ibms -keyalg RSA -keysize 2048 -validity 10000
   ```

2. Add signing config to `local.properties`:
   ```properties
   KEYSTORE_PATH=/path/to/ibms.keystore
   KEYSTORE_PASSWORD=yourpassword
   KEY_ALIAS=ibms
   KEY_PASSWORD=yourpassword
   ```

3. Add signing config to `app/build.gradle.kts` under `signingConfigs` and reference it in `buildTypes.release`.

---

## Architecture

```
app/
  src/main/java/com/ispcms/
    data/
      api/
        ApiService.kt          Retrofit interface — all endpoints
        AuthInterceptor.kt     Attaches Bearer token to every request
        TokenAuthenticator.kt  OkHttp authenticator — refreshes token on 401,
                               clears session if refresh also fails → forces re-login
      local/
        SecureStorage.kt       EncryptedSharedPreferences + Android Keystore
                               Stores access token, refresh token, cached user
                               Passwords are NEVER stored
      models/
        Models.kt              Kotlin data classes matching backend JSON
      repository/
        AuthRepository.kt      Login, logout, refresh current user
        DashboardRepository.kt Dashboard stats (prefix-scoped by backend)
        AccountRepository.kt   Internet accounts with search
        BillRepository.kt      Bills, account due, collect payment
        ReportRepository.kt    Collection report
        ExpenseRepository.kt   Expenses + categories
    di/
      NetworkModule.kt         Hilt module — OkHttpClient, Retrofit, ApiService
    ui/
      auth/                    Login screen + ViewModel
      dashboard/               Dashboard screen + ViewModel
      accounts/                Internet Accounts screen + ViewModel
      bills/                   Bills screen + collect dialog + ViewModel
      reports/                 Collection Report screen + ViewModel
      expenses/                Expenses screen + add dialog + ViewModel
      profile/                 Profile + logout screen
      main/                    Bottom nav shell (MainScreen, MainViewModel)
      navigation/              NavGraph, Screen sealed class
      components/              Reusable StatCard, ErrorView, LoadingView, etc.
      theme/                   Material3 color scheme, typography
    utils/
      UiState.kt               sealed class Loading | Success | Error | Idle
      PermissionHelper.kt      Evaluates module:action permissions for UI gating
```

---

## Authentication Flow

```
User enters credentials
        ↓
POST /auth/login
        ↓
Access token + Refresh token saved to EncryptedSharedPreferences (Android Keystore)
User object (with role + permissions) cached
        ↓
Dashboard opens — permissions drive tab visibility
        ↓
On every API call → AuthInterceptor adds Authorization: Bearer <access_token>
        ↓
If 401 returned → TokenAuthenticator tries POST /auth/refresh
        ↓
  Success → new access token saved, request retried automatically
  Failure → session cleared → app redirects to Login
```

---

## Security

| Concern | How addressed |
|---------|---------------|
| Token storage | `EncryptedSharedPreferences` with AES-256-GCM MasterKey in Android Keystore |
| Password storage | Never stored — only tokens |
| Cleartext HTTP | Blocked by `network_security_config.xml` for all non-local hosts |
| Authorization | All enforced on the backend; app only hides/shows UI |
| Token refresh | Handled transparently by OkHttp `Authenticator` |
| Release builds | ProGuard minification + resource shrinking |

---

## Key Dependencies

| Library | Version | Purpose |
|---------|---------|---------|
| Jetpack Compose BOM | 2024.04.01 | All Compose UI |
| Hilt | 2.51.1 | Dependency injection |
| Retrofit | 2.11.0 | HTTP client |
| OkHttp | 4.12.0 | Networking + logging |
| Gson | 2.10.1 | JSON serialization |
| security-crypto | 1.1.0-alpha06 | EncryptedSharedPreferences |
| Navigation Compose | 2.7.7 | In-app navigation |
| Coil | 2.6.0 | Image loading |
