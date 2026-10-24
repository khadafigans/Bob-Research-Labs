# THE GRIM REAPER - AWS Bedrock Setup Guide

## Overview
THE GRIM REAPER uses AWS Bedrock to run Claude 3.5 Sonnet for autonomous penetration testing.

## Prerequisites
- AWS Account with Bedrock access enabled
- AWS Access Key ID and Secret Access Key with Bedrock permissions
- Claude 3.5 Sonnet model access enabled in your AWS region

## Quick Setup

### Option 1: Automated Script (Recommended)
```bash
cd /var/reaper-full
export AWS_ACCESS_KEY_ID='your_access_key_here'
export AWS_SECRET_ACCESS_KEY='your_secret_key_here'
export AWS_REGION='us-east-1'
./setup-bedrock.sh
```

### Option 2: Manual API Configuration
```bash
curl -X POST http://localhost:8787/api/llm \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "bedrock",
    "model": "anthropic.claude-3-5-sonnet-20241022-v2:0",
    "base_url": "https://bedrock-runtime.us-east-1.amazonaws.com",
    "api_key": "ACCESS_KEY:SECRET_KEY:REGION",
    "rate_per_minute": 50,
    "context_window_k": 200
  }'
```

### Option 3: Via Web Interface
1. Navigate to `http://localhost:8787/system/settings`
2. Go to "LLM 설정" (LLM Configuration)
3. Fill in:
   - **Provider**: `bedrock`
   - **Model**: `anthropic.claude-3-5-sonnet-20241022-v2:0`
   - **Base URL**: `https://bedrock-runtime.us-east-1.amazonaws.com`
   - **API Key**: Format: `ACCESS_KEY_ID:SECRET_ACCESS_KEY:REGION`
   - **Rate Limit**: `50` requests/minute
4. Click "Test Connection" then "Save"

## AWS Bedrock API Key Format
```
AWS_ACCESS_KEY_ID:AWS_SECRET_ACCESS_KEY:AWS_REGION
```

Example:
```
AKIAIOSFODNN7EXAMPLE:wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY:us-east-1
```

## Supported AWS Regions
- `us-east-1` (N. Virginia) - Recommended
- `us-west-2` (Oregon)
- `eu-west-1` (Ireland)
- `ap-southeast-1` (Singapore)
- `ap-northeast-1` (Tokyo)

## Available Claude Models on Bedrock
- `anthropic.claude-3-5-sonnet-20241022-v2:0` (Latest, Recommended)
- `anthropic.claude-3-5-sonnet-20240620-v1:0`
- `anthropic.claude-3-opus-20240229-v1:0`
- `anthropic.claude-3-sonnet-20240229-v1:0`
- `anthropic.claude-3-haiku-20240307-v1:0`

## Verification
Check if configuration is active:
```bash
curl http://localhost:8787/api/llm | python3 -m json.tool
```

Expected output:
```json
{
  "provider": "bedrock",
  "model": "anthropic.claude-3-5-sonnet-20241022-v2:0",
  "base_url": "https://bedrock-runtime.us-east-1.amazonaws.com",
  "rate_per_minute": 50,
  "context_window_k": 200
}
```

## Troubleshooting

### Error: "provider init failed"
- Verify AWS credentials are correct
- Check that Bedrock is enabled in your AWS account
- Ensure Claude model access is approved in AWS console

### Error: "AccessDeniedException"
Your IAM user needs these permissions:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "bedrock:InvokeModel",
        "bedrock:InvokeModelWithResponseStream"
      ],
      "Resource": "arn:aws:bedrock:*::foundation-model/anthropic.claude-*"
    }
  ]
}
```

### Error: "ThrottlingException"
- Reduce `rate_per_minute` value
- Request quota increase in AWS Service Quotas

### Logs
Check service logs:
```bash
tail -f /tmp/reaper.log
```

## Cost Estimation
Claude 3.5 Sonnet v2 on Bedrock pricing (as of 2024):
- Input: ~$3 per million tokens
- Output: ~$15 per million tokens

Typical autonomous pentest task:
- Duration: 2-8 hours
- Token usage: 500K-2M tokens
- Estimated cost: $10-50 per task

## Next Steps
After AWS Bedrock is configured:
1. Create your first task in the web interface
2. Set target scope and objectives
3. Monitor autonomous progress
4. Review findings and validate exploits
5. Configure Telegram notifications (optional)

## Support
- Service logs: `/tmp/reaper.log`
- Database: `reaper` on localhost PostgreSQL
- Port: `8787`
- Data directory: `/var/reaper-full/data`
