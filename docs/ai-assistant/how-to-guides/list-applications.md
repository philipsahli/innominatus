# How to List Applications Using AI

**Goal**: View all deployed applications on your platform.

**Time**: 1 minute

## Quick Steps

### Web UI
1. Open AI Assistant at `http://localhost:8081/ai-assistant`
2. Type: `list my applications`
3. Press Enter or click Send

### CLI
```bash
./innominatus-ctl chat --one-shot "list my applications"
```

## Expected Response

```
You have 3 applications:
• demo-app (production) - 5 resources
• test-service (staging) - 3 resources
• api-gateway (production) - 8 resources
```

## Alternative Phrasings

All of these work:
- `list my applications`
- `show me all apps`
- `what applications are deployed`
- `which apps are running`

## Understanding the Output

Each line shows:
- **Name**: Application identifier
- **Environment**: production, staging, development
- **Resource count**: Number of associated resources (databases, caches, volumes, etc.)

## Getting More Details

Follow up with:
```
"tell me more about demo-app"
"show resources for demo-app"
```

## Related Tasks

- [Get Application Details](./get-application-details.md)
- [List Workflows](./list-workflows.md)
- [View Platform Statistics](./view-statistics.md)
