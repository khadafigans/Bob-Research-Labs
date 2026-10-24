#!/bin/bash
# AWS Bedrock Configuration for THE GRIM REAPER
# This script configures Claude 3.5 Sonnet via AWS Bedrock

set -e

echo "=== THE GRIM REAPER - AWS Bedrock Configuration ==="
echo ""

# Check if AWS credentials are available
if [ -z "$AWS_ACCESS_KEY_ID" ] || [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
    echo "⚠️  AWS credentials not found in environment variables"
    echo ""
    echo "Please set:"
    echo "  export AWS_ACCESS_KEY_ID='your_access_key_id'"
    echo "  export AWS_SECRET_ACCESS_KEY='your_secret_access_key'"
    echo "  export AWS_REGION='us-east-1'  # or your preferred region"
    echo ""
    read -p "Do you want to enter them now? (y/n): " answer
    if [ "$answer" = "y" ]; then
        read -p "AWS Access Key ID: " AWS_ACCESS_KEY_ID
        read -sp "AWS Secret Access Key: " AWS_SECRET_ACCESS_KEY
        echo ""
        read -p "AWS Region [us-east-1]: " AWS_REGION
        AWS_REGION=${AWS_REGION:-us-east-1}
        export AWS_ACCESS_KEY_ID
        export AWS_SECRET_ACCESS_KEY
        export AWS_REGION
    else
        echo "Exiting. Please set AWS credentials and run again."
        exit 1
    fi
fi

AWS_REGION=${AWS_REGION:-us-east-1}

echo ""
echo "AWS Configuration:"
echo "  Access Key ID: ${AWS_ACCESS_KEY_ID:0:8}...${AWS_ACCESS_KEY_ID: -4}"
echo "  Region: $AWS_REGION"
echo ""

# Wait for service to be ready
echo "Checking if THE GRIM REAPER service is running..."
if ! curl -s http://localhost:8787/api/health > /dev/null 2>&1; then
    echo "❌ Service not running on port 8787"
    echo "Start it with: cd /var/reaper-full && ./reaper -addr :8787"
    exit 1
fi

echo "✅ Service is running"
echo ""

# Configure AWS Bedrock via API
echo "Configuring AWS Bedrock (Claude 3.5 Sonnet)..."

RESPONSE=$(curl -s -X POST http://localhost:8787/api/llm \
  -H "Content-Type: application/json" \
  -d "{
    \"provider\": \"bedrock\",
    \"model\": \"anthropic.claude-3-5-sonnet-20241022-v2:0\",
    \"base_url\": \"https://bedrock-runtime.$AWS_REGION.amazonaws.com\",
    \"api_key\": \"$AWS_ACCESS_KEY_ID:$AWS_SECRET_ACCESS_KEY:$AWS_REGION\",
    \"rate_per_minute\": 50,
    \"context_window_k\": 200
  }")

if echo "$RESPONSE" | grep -q '"provider"'; then
    echo "✅ AWS Bedrock configured successfully!"
    echo ""
    echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
    echo ""
    echo "=== Configuration Complete ==="
    echo ""
    echo "The GRIM REAPER is now configured with:"
    echo "  Provider: AWS Bedrock"
    echo "  Model: Claude 3.5 Sonnet v2"
    echo "  Region: $AWS_REGION"
    echo "  Rate Limit: 50 requests/minute"
    echo ""
    echo "You can now:"
    echo "  1. Access the web interface: http://localhost:8787"
    echo "  2. Create new penetration testing tasks"
    echo "  3. Monitor findings and results"
    echo ""
else
    echo "❌ Configuration failed!"
    echo "Response: $RESPONSE"
    exit 1
fi
