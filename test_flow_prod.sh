#!/bin/bash

# Configuration
BASE_URL="https://sir.puem.me"
ADMIN_EMAIL="xx@gmail.com"
ADMIN_PASSWORD="xx"

# Generate Random Credentials
CLIENT_ID="client-ad94588c"
CLIENT_SECRET="8c879eaa1125dd435e8155bb442439fa"
CLIENT_NAME="Admin POC ($(date +%Y-%m-%d))"

echo "🚀 Starting Production Setup Flow for $BASE_URL"

# 1. Ask for Setup Secret
read -p "Enter your SETUP_SECRET (Cloudflare secret): " SETUP_SECRET

if [ -z "$SETUP_SECRET" ]; then
    echo "❌ Error: SETUP_SECRET is required."
    exit 1
fi

echo "---"
echo "Step 1: Running /setup on Production..."

RESPONSE=$(curl -s -X POST "$BASE_URL/setup?secret=$SETUP_SECRET" \
     -H "Content-Type: application/json" \
     -d "{
       \"admin_email\": \"$ADMIN_EMAIL\",
       \"admin_password\": \"$ADMIN_PASSWORD\",
       \"client_id\": \"$CLIENT_ID\",
       \"client_secret\": \"$CLIENT_SECRET\",
       \"client_name\": \"$CLIENT_NAME\"
     }")

if echo "$RESPONSE" | grep -q "admin_id"; then
    echo "✅ Setup Successful!"
    echo "Admin User: $ADMIN_EMAIL"
    echo "Client ID: $CLIENT_ID"
    echo "---"
    echo "🎉 Now you can run the POC TUI with these credentials:"
    echo ""
    echo "cd poc"
    echo "CLIENT_ID=\"$CLIENT_ID\" CLIENT_SECRET=\"$CLIENT_SECRET\" go run main.go"
    echo ""
else
    echo "❌ Setup Failed!"
    echo "Response: $RESPONSE"
    exit 1
fi
