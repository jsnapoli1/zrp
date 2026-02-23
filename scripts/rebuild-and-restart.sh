#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "🔄 ZRP Rebuild & Restart Script"
echo "==============================="
echo ""

# Step 1: Run verification first
echo -e "${BLUE}Step 1: Running verification...${NC}"
if ! bash scripts/verify.sh; then
    echo -e "${RED}✗ Verification failed. Not proceeding with rebuild.${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✓ Verification passed!${NC}"
echo ""

# Step 2: Rebuild frontend
echo -e "${BLUE}Step 2: Rebuilding frontend...${NC}"
cd frontend
npm run build
cd ..
echo -e "${GREEN}✓ Frontend rebuilt${NC}"
echo ""

# Step 3: Rebuild backend
echo -e "${BLUE}Step 3: Rebuilding backend...${NC}"
go build -o zrp ./cmd/zrp
echo -e "${GREEN}✓ Backend rebuilt${NC}"
echo ""

# Step 4: Kill existing ZRP server
echo -e "${BLUE}Step 4: Stopping existing ZRP server...${NC}"
if pkill -f "./zrp" 2>/dev/null; then
    echo -e "${GREEN}✓ Stopped existing server${NC}"
    sleep 1
else
    echo -e "${YELLOW}⚠ No running server found${NC}"
fi
echo ""

# Step 5: Start new server
echo -e "${BLUE}Step 5: Starting new ZRP server...${NC}"
./zrp &
ZRP_PID=$!
echo -e "${GREEN}✓ Server started (PID: $ZRP_PID)${NC}"
echo ""

# Step 6: Wait for server to be healthy
echo -e "${BLUE}Step 6: Checking server health...${NC}"
sleep 2

# Try to connect to the server (assumes it runs on port 8080 by default)
MAX_ATTEMPTS=10
ATTEMPT=0
while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if curl -s http://localhost:8080 > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Server is healthy and responding${NC}"
        echo ""
        echo "==============================="
        echo -e "${GREEN}🎉 ZRP successfully rebuilt and restarted!${NC}"
        echo -e "Server PID: ${BLUE}$ZRP_PID${NC}"
        echo -e "Access at: ${BLUE}http://localhost:8080${NC}"
        exit 0
    fi
    ATTEMPT=$((ATTEMPT + 1))
    echo -e "${YELLOW}⏳ Waiting for server... (attempt $ATTEMPT/$MAX_ATTEMPTS)${NC}"
    sleep 1
done

echo -e "${RED}✗ Server failed to respond after $MAX_ATTEMPTS attempts${NC}"
echo -e "${YELLOW}Server may still be starting. Check logs with: tail -f server.log${NC}"
exit 1
