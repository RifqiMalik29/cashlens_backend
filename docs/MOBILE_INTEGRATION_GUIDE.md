# Mobile Integration Guide

This document contains all the necessary information for the mobile team to integrate with the new CashLens backend setup.

## 1. Environment URLs

| Environment | Base URL |
| :--- | :--- |
| **Development** | `http://<YOUR_LOCAL_IP>:8080` (or as configured in `.env`) |
| **Production** | `https://cashlens-backend-552315397645.us-central1.run.app` |

## 2. API Contract (Source of Truth)
The updated API specification is located in `apidog-openapi.yaml` in the root of the backend repository. You can import this file directly into **Apidog**, **Postman**, or **Bruno**.

## 3. Key Changes in API Responses

### Mandatory "data" Wrapper
All successful backend responses are now wrapped in a `data` object for consistency.
*   **Old response:** `{ "id": "...", "email": "..." }`
*   **New response:** `{ "data": { "id": "...", "email": "..." } }`
*   *Note:* If you use Axios, you might need to access it via `response.data.data`.

### Authentication Fields
*   **Access Token:** Renamed from `token` to `access_token`.
*   **Refresh Token:** Now explicitly returned as `refresh_token`.

## 4. Receipt Scanner & OCR Fallback
We have implemented a **Cascading Fallback** strategy to ensure the scanner works even under poor network conditions or with blurry images.

**Endpoint:** `POST /api/v1/receipts/scan`
**Content-Type:** `multipart/form-data`

**Required Setup:**
1.  **Local OCR:** On the mobile app, use Google ML Kit (local) to extract text from the receipt image.
2.  **Request Body:**
    *   `image`: The receipt image file.
    *   `ocr_text`: (Optional) The text string extracted by your local ML Kit.

**How it works:**
*   The backend first tries high-precision AI Vision on the `image`.
*   If vision fails or returns low confidence, the backend automatically falls back to parsing the `ocr_text` you provided.

## 5. Mobile Environment Setup (Expo/React Native)
To support side-by-side installation of Dev and Prod versions, we recommend the following configuration in your `app.config.js`:

```javascript
const IS_DEV = process.env.APP_VARIANT === 'development';

export default {
  expo: {
    name: IS_DEV ? "CashLens (Dev)" : "CashLens",
    scheme: IS_DEV ? "cashlens-dev" : "cashlens",
    extra: {
      baseUrl: IS_DEV ? "http://<YOUR_IP>:8080" : "https://cashlens-backend-552315397645.us-central1.run.app",
    },
    android: {
      package: IS_DEV ? "com.cashlens.app.dev" : "com.cashlens.app"
    },
    ios: {
      bundleIdentifier: IS_DEV ? "com.cashlens.app.dev" : "com.cashlens.app"
    }
  }
};
```

## 6. New Features Now Available
*   **Email Verification (OTP):**
    *   `POST /api/v1/auth/confirm`: Submit `{ "email": "...", "otp": "123456" }` to verify account.
    *   `POST /api/v1/auth/resend-confirmation`: Submit `{ "email": "..." }` to get a new code.
*   **Telegram Linking:** `GET /api/v1/auth/telegram/status` to check if the user is linked.
*   **Subscription Status:** `GET /api/v1/subscription` returns tier, expiry, and detailed usage quotas.
*   **Language Settings:** `PATCH /api/v1/auth/language` (supports `id` and `en`).
*   **Push Notifications:** `PATCH /api/v1/auth/push-token` to register the device token.
*   **Account Deletion (Mandatory for App Stores):**
    *   `DELETE /api/v1/auth/me`: Permanently deletes the user account and all data. This must be accessible from within the app settings.
