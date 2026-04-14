# GCP & Neon Migration Guide

This document outlines the steps to migrate the CashLens backend from Railway to **Google Cloud Platform (GCP)** and the database to **Neon**.

## Infrastructure Components

- **Compute:** [Google Cloud Run](https://cloud.google.com/run) (Serverless Docker)
- **Database:** [Neon PostgreSQL](https://neon.tech/) (Serverless Postgres)
- **CI/CD:** [Google Cloud Build](https://cloud.google.com/build)
- **Secrets:** [Google Secret Manager](https://cloud.google.com/secret-manager)
- **Artifacts:** [Google Artifact Registry](https://cloud.google.com/artifact-registry)

---

## 1. Neon Database Setup

1. Create a project at [Neon.tech](https://neon.tech/).
2. Create a new database (e.g., `cashlens`).
3. Copy the **Connection String** (use the pooled version if available).
   - Format: `postgresql://user:password@ep-hostname.region.aws.neon.tech/neondb?sslmode=require`
4. Test the connection locally:
   ```bash
   DATABASE_URL="your_neon_url" make migrate-up
   ```

---

## 2. GCP Project Initialization

Run these commands using the `gcloud` CLI to enable required services:

```bash
# Set your project ID
export PROJECT_ID="your-project-id"
gcloud config set project $PROJECT_ID

# Enable APIs
gcloud services enable \
    run.googleapis.com \
    cloudbuild.googleapis.com \
    artifactregistry.googleapis.com \
    secretmanager.googleapis.com

# Create Artifact Registry repository
gcloud artifacts repositories create cashlens-repo \
    --repository-format=docker \
    --location=us-central1 \
    --description="Docker repository for CashLens"
```

---

## 3. Secret Manager Configuration

You must create the following secrets in [Secret Manager](https://console.cloud.google.com/security/secret-manager). These match the variables defined in `cloudbuild.yaml`.

| Secret Name | Description |
| :--- | :--- |
| `DATABASE_URL` | Neon connection string |
| `JWT_SECRET` | Secure random string for JWT signing |
| `GEMINI_API_KEY` | Google Gemini API key |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token |
| `XENDIT_SECRET_KEY` | Xendit API secret key |
| `XENDIT_WEBHOOK_TOKEN` | Xendit callback verification token |

**Example command to create a secret:**
```bash
echo -n "your-secret-value" | gcloud secrets create DATABASE_URL --data-file=-
```

---

## 4. IAM Permissions

Grant the Cloud Build Service Account permission to deploy to Cloud Run and access secrets.

```bash
# Get your project number
export PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format='value(projectNumber)')

# Grant Cloud Run Admin
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$PROJECT_NUMBER@cloudbuild.gserviceaccount.com" \
    --role="roles/run.admin"

# Grant IAM Service Account User (needed to act as the runtime service account)
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$PROJECT_NUMBER@cloudbuild.gserviceaccount.com" \
    --role="roles/iam.serviceAccountUser"

# Grant Secret Manager Accessor
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$PROJECT_NUMBER@cloudbuild.gserviceaccount.com" \
    --role="roles/secretmanager.secretAccessor"

# Grant Artifact Registry Writer
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$PROJECT_NUMBER@cloudbuild.gserviceaccount.com" \
    --role="roles/artifactregistry.writer"
```

---

## 5. Automation: Cloud Build

The configuration is stored in `cloudbuild.yaml` in the project root.

### Pipeline Steps:
1. **Build:** Build the Docker image from `Dockerfile`.
2. **Push:** Push the image to Artifact Registry.
3. **Deploy:** Deploy the new image to Cloud Run, injecting secrets from Secret Manager.

### Continuous Deployment (GitHub):
1. Go to [Cloud Build Triggers](https://console.cloud.google.com/cloud-build/triggers).
2. Click **Manage Repositories** and connect your GitHub repo.
3. Create a **Trigger**:
   - **Event:** Push to a branch
   - **Branch:** `^main$`
   - **Configuration:** Cloud Build configuration file (yaml or json)
   - **Location:** `cloudbuild.yaml`

---

## 6. Verification

After the first automated deployment:
1. Check the Cloud Run URL (provided in the deployment logs).
2. Verify the health check: `https://your-cloud-run-url/health`
3. The server automatically runs migrations on startup, so your Neon database schema will stay up to date.
