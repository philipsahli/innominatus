# AI Assistant API Reference

**Purpose**: Complete HTTP API reference for the AI assistant service.

## Base URL

```
http://localhost:8081
```

## Authentication

All AI endpoints require authentication via Bearer token:

```
Authorization: Bearer <api-token>
```

**How to Get Token**:
- **Web UI**: Automatic (session cookie)
- **CLI**: Set `IDP_API_KEY` environment variable
- **API**: Generate via `/api/profile/api-keys` endpoint

## Endpoints

---

### POST /api/ai/chat

**Description**: Send a message to the AI assistant and receive a response.

**Request**:
```http
POST /api/ai/chat HTTP/1.1
Host: localhost:8081
Authorization: Bearer <token>
Content-Type: application/json

{
  "message": "list my applications",
  "context": "production-env"  // Optional
}
```

**Request Body**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `message` | string | Yes | User's question or request |
| `context` | string | No | Additional context (app name, workflow ID, etc.) |

**Response**:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "message": "You have 3 applications:\n• demo-app (production) - 5 resources\n• test-service (staging) - 3 resources\n• api-gateway (production) - 8 resources",
  "generated_spec": "",
  "citations": [
    "docs/FEATURES.md",
    "docs/README.md"
  ],
  "tokens_used": 456,
  "timestamp": "2025-10-06T16:30:00Z"
}
```

**Response Body**:
| Field | Type | Description |
|-------|------|-------------|
| `message` | string | AI's response text (markdown formatted) |
| `generated_spec` | string | YAML Score spec if generated (empty otherwise) |
| `citations` | array | Documentation sources used |
| `tokens_used` | integer | Total tokens consumed (input + output) |
| `timestamp` | string | Response timestamp (ISO 8601) |

**Status Codes**:
- `200 OK`: Success
- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Missing or invalid auth token
- `500 Internal Server Error`: AI service error

**Example with Tool Calling**:

```bash
curl -X POST http://localhost:8081/api/ai/chat \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "show me recent workflows"
  }'
```

Response:
```json
{
  "message": "Recent workflows:\n1. deploy-app (demo-app) - completed (2m 34s)\n2. db-lifecycle (api-gateway) - running (1m 12s)\n3. ephemeral-env (test-env) - completed (45s)",
  "citations": [],
  "tokens_used": 523,
  "timestamp": "2025-10-06T16:30:15Z"
}
```

**Example with Spec Generation**:

```bash
curl -X POST http://localhost:8081/api/ai/chat \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "generate a score spec for a node.js app with postgres"
  }'
```

Response:
```json
{
  "message": "Here's a Score specification for a Node.js application with PostgreSQL:\n\n[explanation text]",
  "generated_spec": "apiVersion: score.dev/v1b1\nmetadata:\n  name: nodejs-app\n...",
  "citations": ["docs/FEATURES.md"],
  "tokens_used": 789,
  "timestamp": "2025-10-06T16:30:30Z"
}
```

---

### POST /api/ai/generate-spec

**Description**: Generate a Score specification from a text description.

**Request**:
```http
POST /api/ai/generate-spec HTTP/1.1
Host: localhost:8081
Authorization: Bearer <token>
Content-Type: application/json

{
  "description": "python fastapi application with postgres database and redis cache",
  "metadata": {
    "team": "backend",
    "environment": "production"
  }
}
```

**Request Body**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `description` | string | Yes | Natural language description of the application |
| `metadata` | object | No | Additional metadata (team, environment, etc.) |

**Response**:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "spec": "apiVersion: score.dev/v1b1\nmetadata:\n  name: fastapi-app\n...",
  "explanation": "This Score specification defines a Python FastAPI application with...",
  "citations": [
    "docs/FEATURES.md",
    "docs/examples/python-app.md"
  ],
  "tokens_used": 1234
}
```

**Response Body**:
| Field | Type | Description |
|-------|------|-------------|
| `spec` | string | Complete YAML Score specification |
| `explanation` | string | AI explanation of the spec components |
| `citations` | array | Knowledge base sources used |
| `tokens_used` | integer | Total tokens consumed |

**Status Codes**:
- `200 OK`: Success
- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Missing or invalid auth token
- `500 Internal Server Error`: Spec generation failed

**Example**:

```bash
curl -X POST http://localhost:8081/api/ai/generate-spec \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "java spring boot app with 2GB memory and mysql database"
  }'
```

---

### GET /api/ai/status

**Description**: Check AI service status and configuration.

**Request**:
```http
GET /api/ai/status HTTP/1.1
Host: localhost:8081
```

**Authentication**: Not required

**Response**:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "enabled": true,
  "llm_provider": "anthropic",
  "embedding_model": "openai",
  "documents_loaded": 25,
  "status": "ready",
  "message": ""
}
```

**Response Body**:
| Field | Type | Description |
|-------|------|-------------|
| `enabled` | boolean | Whether AI service is enabled |
| `llm_provider` | string | LLM provider (`anthropic`) |
| `embedding_model` | string | Embedding provider (`openai`) |
| `documents_loaded` | integer | Number of documents in knowledge base |
| `status` | string | Service status (`ready`, `not_configured`, `error`) |
| `message` | string | Additional status information or error message |

**Status Values**:
- `ready`: AI service is operational
- `not_configured`: Missing required API keys
- `error`: Service initialization or runtime error

**Example**:

```bash
curl http://localhost:8081/api/ai/status
```

---

## Error Responses

All endpoints return errors in a consistent format:

```json
{
  "error": "AI service is not enabled",
  "code": "SERVICE_DISABLED",
  "details": "Missing required environment variables: ANTHROPIC_API_KEY"
}
```

**Common Error Codes**:
- `SERVICE_DISABLED`: AI service not enabled
- `INVALID_REQUEST`: Malformed request body
- `UNAUTHORIZED`: Missing or invalid authentication
- `TOOL_EXECUTION_FAILED`: Tool call to platform API failed
- `LLM_ERROR`: LLM provider API error
- `RAG_ERROR`: Knowledge base retrieval error

## Rate Limiting

**Current Implementation**: No rate limiting

**Recommendation for Production**:
- Implement per-user rate limits (e.g., 30 requests/minute)
- Use token-bucket or sliding window algorithm
- Return `429 Too Many Requests` when exceeded

## Request/Response Format

### Content Types

**Request**: `application/json`
**Response**: `application/json`

### Character Encoding

**UTF-8** for all requests and responses

### Markdown in Responses

AI responses use GitHub-flavored markdown:
- **Bold**: `**text**`
- **Italic**: `*text*`
- **Code**: `` `code` ``
- **Lists**: `• item` or `1. item`
- **Code Blocks**: ` ```yaml ... ``` `

## WebSocket Support

**Status**: Not currently supported

**Future Consideration**: Streaming responses for real-time chat

## CORS Configuration

**Current**: Not configured (same-origin only)

**For Cross-Origin Requests**: Configure CORS headers in server

## Examples

### Complete Chat Flow

```bash
# 1. Check AI status
curl http://localhost:8081/api/ai/status

# 2. List applications
curl -X POST http://localhost:8081/api/ai/chat \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "list my applications"}'

# 3. Get application details
curl -X POST http://localhost:8081/api/ai/chat \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "tell me about demo-app"}'

# 4. Generate spec
curl -X POST http://localhost:8081/api/ai/generate-spec \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"description": "node.js app with postgres"}'

# 5. Deploy generated spec
curl -X POST http://localhost:8081/api/ai/chat \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "deploy this spec:\n<paste yaml here>"}'
```

### Error Handling Example

```javascript
async function askAI(message) {
  try {
    const response = await fetch('http://localhost:8081/api/ai/chat', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ message })
    });

    if (!response.ok) {
      const error = await response.json();
      console.error('AI Error:', error.error);
      return null;
    }

    const data = await response.json();
    console.log('AI Response:', data.message);
    return data;

  } catch (err) {
    console.error('Network Error:', err);
    return null;
  }
}
```

## Integration Examples

### Backstage Plugin

```typescript
// Backstage custom action
import { createTemplateAction } from '@backstage/plugin-scaffolder-backend';

export const askInnominatusAI = createTemplateAction({
  id: 'innominatus:ai:chat',
  async handler(ctx) {
    const response = await fetch(`${ctx.input.baseUrl}/api/ai/chat`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${ctx.input.token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        message: ctx.input.message
      })
    });

    const data = await response.json();
    ctx.output('response', data.message);
  }
});
```

### Python Client

```python
import requests

class InnominatusAI:
    def __init__(self, base_url, api_token):
        self.base_url = base_url
        self.api_token = api_token

    def chat(self, message, context=None):
        url = f"{self.base_url}/api/ai/chat"
        headers = {
            "Authorization": f"Bearer {self.api_token}",
            "Content-Type": "application/json"
        }
        payload = {"message": message}
        if context:
            payload["context"] = context

        response = requests.post(url, json=payload, headers=headers)
        response.raise_for_status()
        return response.json()

    def generate_spec(self, description, metadata=None):
        url = f"{self.base_url}/api/ai/generate-spec"
        headers = {
            "Authorization": f"Bearer {self.api_token}",
            "Content-Type": "application/json"
        }
        payload = {"description": description}
        if metadata:
            payload["metadata"] = metadata

        response = requests.post(url, json=payload, headers=headers)
        response.raise_for_status()
        return response.json()

# Usage
ai = InnominatusAI("http://localhost:8081", "your-api-token")
result = ai.chat("list my applications")
print(result["message"])
```

## See Also

- [Tools Reference](./tools-reference.md) - AI tools for platform interaction
- [Configuration Reference](./configuration.md) - Environment variables and settings
- [CLI Reference](./cli-reference.md) - Command-line interface
