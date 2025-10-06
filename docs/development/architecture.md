# Architecture

innominatus system architecture.

---

## Overview

```mermaid
graph LR
    A[Client] --> B[API Server]
    B --> C[Workflow Executor]
    C --> D[Kubernetes]
    C --> E[Terraform]
    C --> F[Ansible]
    B --> G[(PostgreSQL)]
```

---

## Components

- **API Server**: RESTful API endpoints
- **Workflow Executor**: Orchestrates multi-step workflows
- **Database**: PostgreSQL for persistence
- **Executors**: Kubernetes, Terraform, Ansible integrations

---

**More details coming soon...**
