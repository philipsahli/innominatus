# Testing Guide

Run tests for innominatus.

---

## Unit Tests

```bash
go test ./...
```

---

## Integration Tests

```bash
go test ./internal/... -tags=integration
```

---

## Web UI Tests

```bash
cd web-ui
npm run test
```

---

**More details coming soon...**
