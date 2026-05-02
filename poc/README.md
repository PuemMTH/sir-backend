# OAuth 2.0 Proof of Concept (POC)

This directory contains a Terminal User Interface (TUI) client written in Go that demonstrates the OAuth 2.0 Authorization Code Flow (RFC 8252) against the `sir-backend` production server.

## Prerequisites

Before running the client, you must ensure your production Cloudflare Worker is configured and initialized.

### 1. Set the Setup Secret

The `/setup` endpoint on the production server is protected by a `SETUP_SECRET`. You must define this secret in your Cloudflare environment.

Run the following command in the root of the project:

```bash
npx wrangler secret put SETUP_SECRET --name sir-backend
```

When prompted, enter a secure, random string (e.g., `62c7c7ab3b3ee6689629122daa39a178`). **Remember this secret.**

### 2. (Optional) Clean Existing Data

If you have previously attempted setup and encountered issues like `access_denied`, it is best to start fresh. Run this from the project root:

```bash
npx wrangler d1 execute sir-db --remote --command "DELETE FROM users; DELETE FROM oauth_clients; DELETE FROM auth_codes; DELETE FROM refresh_tokens;"
```

### 3. Run the Production Setup Script

Run the automated setup script to create the initial admin user and register a new OAuth client with randomized credentials.

```bash
./test_flow_prod.sh
```

- When prompted for the `SETUP_SECRET`, paste the string you created in Step 1.
- The script will output a success message along with the dynamically generated `CLIENT_ID` and `CLIENT_SECRET`.
- **Copy the final `go run` command provided by the script.**

## Running the TUI Client

1. Navigate to this `poc` directory:
   ```bash
   cd poc
   ```

2. Paste and run the command provided by the setup script. It will look similar to this:
   ```bash
   CLIENT_ID="client-xxxxxxxx" CLIENT_SECRET="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" BASE_URL="https://sir.puem.me" go run main.go
   ```

3. **Authentication:**
   - Press `Enter` in the terminal. Your default web browser will open automatically.
   - Log in using the admin credentials:
     - **Email:** `admin@gmail.com`
     - **Password:** `admin`
   - After successful login, you can close the browser tab.

4. **Result:**
   - The TUI will capture the authorization code, exchange it for an Access Token and Refresh Token, and then fetch data from the protected `/api/me` and `/api/notes` endpoints.
   - The results will be displayed beautifully formatted in your terminal.
