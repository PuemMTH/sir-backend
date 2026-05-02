#!/bin/bash

# Configuration
API_URL="https://sir.puem.me/"
SETUP_SECRET="dev-secret-123"
ADMIN_EMAIL="admin@example.com"
ADMIN_PASS="password123"
CLIENT_ID="test-client"
CLIENT_SECRET="client-secret-456"
REDIRECT_URI="http://127.0.0.1:8080/callback"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting OAuth 2.0 Flow Test...${NC}"

# 1. Health Check
echo -n "Checking API health... "
HEALTH=$(curl -s "$API_URL/health" | grep -o "ok")
if [ "$HEALTH" == "ok" ]; then
    echo -e "${GREEN}OK${NC}"
else
    echo -e "${RED}FAILED${NC} (Is 'wrangler dev' running?)"
    exit 1
fi

# 2. Initial Setup (Seeder)
echo -n "Running initial setup... "
SETUP_RES=$(curl -s -X POST "$API_URL/setup?secret=$SETUP_SECRET" \
    -H "Content-Type: application/json" \
    -d "{
        \"admin_email\": \"$ADMIN_EMAIL\",
        \"admin_password\": \"$ADMIN_PASS\",
        \"client_id\": \"$CLIENT_ID\",
        \"client_secret\": \"$CLIENT_SECRET\",
        \"client_name\": \"Integration Test Client\"
    }")

if echo "$SETUP_RES" | grep -q "admin_id"; then
    echo -e "${GREEN}SUCCESS${NC}"
elif echo "$SETUP_RES" | grep -q "already_initialized"; then
    echo -e "${GREEN}ALREADY INITIALIZED${NC}"
else
    echo -e "${RED}FAILED${NC}"
    echo "Response: $SETUP_RES"
    exit 1
fi

# 3. Simulate Login & Authorization
echo -n "Simulating login and obtaining auth code... "
# We simulate the POST to /oauth/authorize that the login form would do
# We need to capture the redirect location to get the code
AUTH_RES=$(curl -s -i -X POST "$API_URL/oauth/authorize" \
    -d "client_id=$CLIENT_ID" \
    -d "redirect_uri=$REDIRECT_URI" \
    -d "state=random-state-123" \
    -d "scope=openid profile" \
    -d "email=$ADMIN_EMAIL" \
    -d "password=$ADMIN_PASS")

AUTH_CODE=$(echo "$AUTH_RES" | grep -oE "code=[^& ]+" | cut -d= -f2 | tr -d '\r')

if [ -n "$AUTH_CODE" ]; then
    echo -e "${GREEN}CODE OBTAINED${NC} ($AUTH_CODE)"
else
    echo -e "${RED}FAILED${NC} to obtain auth code"
    echo "Response header check: $AUTH_RES"
    exit 1
fi

# 4. Exchange Code for Token
echo -n "Exchanging code for access token... "
TOKEN_RES=$(curl -s -X POST "$API_URL/oauth/token" \
    -d "grant_type=authorization_code" \
    -d "code=$AUTH_CODE" \
    -d "redirect_uri=$REDIRECT_URI" \
    -d "client_id=$CLIENT_ID" \
    -d "client_secret=$CLIENT_SECRET")

ACCESS_TOKEN=$(echo "$TOKEN_RES" | grep -oE '"access_token":"[^"]+"' | cut -d'"' -f4)
REFRESH_TOKEN=$(echo "$TOKEN_RES" | grep -oE '"refresh_token":"[^"]+"' | cut -d'"' -f4)

if [ -n "$ACCESS_TOKEN" ]; then
    echo -e "${GREEN}TOKEN RECEIVED${NC}"
else
    echo -e "${RED}FAILED${NC}"
    echo "Response: $TOKEN_RES"
    exit 1
fi

# 5. Access Protected Resource (/api/me)
echo -n "Testing protected resource /api/me... "
ME_RES=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" "$API_URL/api/me")

if echo "$ME_RES" | grep -q "$ADMIN_EMAIL"; then
    echo -e "${GREEN}SUCCESS${NC}"
    echo "Profile: $ME_RES"
else
    echo -e "${RED}FAILED${NC}"
    echo "Response: $ME_RES"
    exit 1
fi

# 5.1 Test Resource Access (Notes)
echo -n "Testing private resource CRUD (Notes)... "
NOTE_RES=$(curl -s -X POST "$API_URL/api/notes" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"title":"Test Note","content":"This is a private note."}')

NOTE_ID=$(echo "$NOTE_RES" | grep -oE '"id":"[^"]+"' | cut -d'"' -f4)
if [ -z "$NOTE_ID" ]; then
    # Fallback to uppercase ID if needed, though GORM/JSON usually uses lowercase if tagged
    NOTE_ID=$(echo "$NOTE_RES" | grep -oE '"ID":"[^"]+"' | cut -d'"' -f4)
fi

if [ -n "$NOTE_ID" ]; then
    LIST_RES=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" "$API_URL/api/notes")
    if echo "$LIST_RES" | grep -q "$NOTE_ID"; then
        echo -e "${GREEN}SUCCESS${NC} (Note $NOTE_ID created and verified)"
    else
        echo -e "${RED}FAILED${NC} (Note created but not found in list)"
        exit 1
    fi
else
    echo -e "${RED}FAILED${NC} (Could not create note)"
    echo "Response: $NOTE_RES"
    exit 1
fi

# 6. Test Token Refresh
echo -n "Testing refresh token rotation (waiting 1s)... "
sleep 1
REFRESH_RES=$(curl -s -X POST "$API_URL/oauth/token" \
    -d "grant_type=refresh_token" \
    -d "refresh_token=$REFRESH_TOKEN" \
    -d "client_id=$CLIENT_ID" \
    -d "client_secret=$CLIENT_SECRET")

NEW_ACCESS_TOKEN=$(echo "$REFRESH_RES" | grep -oE '"access_token":"[^"]+"' | cut -d'"' -f4)

if [ -n "$NEW_ACCESS_TOKEN" ] && [ "$NEW_ACCESS_TOKEN" != "$ACCESS_TOKEN" ]; then
    echo -e "${GREEN}SUCCESS${NC}"
else
    echo -e "${RED}FAILED${NC}"
    echo "Response: $REFRESH_RES"
    exit 1
fi

echo -e "\n${GREEN}All tests passed!${NC}"
