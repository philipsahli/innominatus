# Documentation Migration - Quick Wins Plan

**Focus:** Maximize impact, minimize effort
**Timeline:** 1 Woche (5 Arbeitstage)
**Goal:** 80% der DevEx-Probleme lösen mit 20% Aufwand

---

## Quick Wins Overview

| # | Quick Win | Effort | Impact | ROI | Prompt |
|---|-----------|--------|--------|-----|--------|
| 1 | README.md Persona Split | 1 Tag | CRITICAL | ⭐⭐⭐⭐⭐ | [Link](#prompt-1) |
| 2 | User Guide - Getting Started | 1 Tag | HIGH | ⭐⭐⭐⭐⭐ | [Link](#prompt-2) |
| 3 | CLI Help Separation | 1 Tag | MEDIUM | ⭐⭐⭐⭐ | [Link](#prompt-3) |
| 4 | Recipe: Node.js + PostgreSQL | 1 Tag | HIGH | ⭐⭐⭐⭐⭐ | [Link](#prompt-4) |
| 5 | Platform Quick Install | 1 Tag | HIGH | ⭐⭐⭐⭐ | [Link](#prompt-5) |

**Total:** 5 Tage = 1 Woche Sprint

---

## The Problem (CRITICAL)

### Aktuelle Situation: 🔴 BROKEN

```
README.md (870 Zeilen) - VERMISCHT:

Platform Team Stuff:
├── Build from Source
├── Docker Image
├── Kubernetes Deployment
├── Database Configuration
├── Production Setup
└── Security Hardening

Platform User Stuff:
├── Deploy Application
├── Monitor Workflows
└── CLI Usage

Contributor Stuff:
├── Build & Test
└── Contributing

Result: CONFUSION für alle drei Zielgruppen!
```

### Developer Experience Today:

```
Developer öffnet README → Liest "Build from Source"
→ "Oh nein, ich muss Go installieren? PostgreSQL? Das ist zu komplex!"
→ GIBT AUF

Dabei: Platform Team hat schon alles aufgesetzt!
Developer muss nur: URL + API Key + CLI
```

**Impact:**
- Time to First Success: >2 Stunden (sollte <15 Min sein)
- Developer gibt auf vor erstem Deployment
- Support Tickets: 50/Woche "Wie fange ich an?"

---

## Solution: Persona-Driven Documentation

### Three Separate Paths:

```
Landing (README.md)
│
├─→ 🧑‍💻 Platform User Path
│   Goal: Deploy my app
│   Time: 15 minutes to first success
│   Needs: Getting Started, Recipes, Troubleshooting
│
├─→ ⚙️ Platform Team Path
│   Goal: Operate innominatus
│   Time: 4 hours to production
│   Needs: Installation, Configuration, Operations
│
└─→ 🔨 Contributor Path
    Goal: Develop innominatus
    Time: 30 minutes to build
    Needs: Build Guide, Architecture, Testing
```

---

## Quick Win Details

### <a name="prompt-1"></a>Quick Win #1: README.md Persona Split

**File:** `.claude/prompts/quickwin-1-readme-persona-split.md`

**Aufgabe:**
- Baue README.md um auf max 300 Zeilen (aktuell 870!)
- Erste 100 Zeilen: Klare Persona-Auswahl
- Verschiebe technische Details in jeweilige Guides

**Outcome:**
```markdown
# innominatus

## 👋 Choose Your Path:

┌────────────────────┬──────────────────────┐
│  🧑‍💻 I'm a Developer │  ⚙️ I'm Platform Team  │
│  [User Guide]      │  [Platform Guide]    │
└────────────────────┴──────────────────────┘
```

**Success Metric:**
- Developer sieht sofort: "Platform Team has set up for you"
- Keine Verwirrung mehr

**Prompt:** [.claude/prompts/quickwin-1-readme-persona-split.md](.claude/prompts/quickwin-1-readme-persona-split.md)

---

### <a name="prompt-2"></a>Quick Win #2: User Guide - Getting Started

**File:** `.claude/prompts/quickwin-2-user-guide-getting-started.md`

**Aufgabe:**
- Erstelle vollständigen "Getting Started" für Platform Users
- Focus: "Your Platform Team has set up innominatus"
- Time to First Success: <15 Minuten

**Outcome:**
```
docs/user-guide/getting-started.md

Step 1: Get Platform Access (2 min)
  → Find platform URL
  → Get API key

Step 2: Install CLI (2 min)
  → brew install innominatus-cli

Step 3: Deploy First App (5 min)
  → innominatus-ctl deploy hello-world.yaml
  → ✅ SUCCESS! App is live!

Next Steps: Deploy real app →
```

**Success Metric:**
- User deployed app in 15 Minuten
- "SUCCESS!" Moment klar und motivierend

**Prompt:** [.claude/prompts/quickwin-2-user-guide-getting-started.md](.claude/prompts/quickwin-2-user-guide-getting-started.md)

---

### <a name="prompt-3"></a>Quick Win #3: CLI Help Separation

**File:** `.claude/prompts/quickwin-3-cli-help-separation.md`

**Aufgabe:**
- Trenne CLI Help in User Mode (default) vs Admin Mode
- Reduziere Commands von 25+ auf 9 (User Mode)

**Outcome:**
```bash
# Default (User Mode)
./innominatus-ctl --help

COMMON COMMANDS (9 commands)
  deploy, status, logs, delete
  list, list-goldenpaths
  tutorial, examples, docs

→ Advanced: --advanced --help
→ Admin: --admin --help

# Instead of 25+ mixed commands
```

**Success Metric:**
- Cognitive Load: -70%
- "I don't know what command to use" → "Oh, I'll run tutorial!"

**Prompt:** [.claude/prompts/quickwin-3-cli-help-separation.md](.claude/prompts/quickwin-3-cli-help-separation.md)

---

### <a name="prompt-4"></a>Quick Win #4: Recipe - Node.js + PostgreSQL

**File:** `.claude/prompts/quickwin-4-recipe-nodejs-postgres.md`

**Aufgabe:**
- Erstelle vollständige, **copy-paste ready** Recipe
- Häufigstes Szenario: REST API mit Database

**Outcome:**
```
docs/user-guide/recipes/nodejs-postgres.md
examples/nodejs-postgres/
├── score.yaml          ← Complete, working example
├── app/server.js       ← Express API with DB
├── app/Dockerfile      ← Production-ready
└── README.md

Features:
- PostgreSQL database (managed)
- Redis cache
- Health checks
- Monitoring
- Auto-scaling
```

**Success Metric:**
- User kann kopieren + anpassen in 15 Min
- Kein "How do I use database?" mehr

**Prompt:** [.claude/prompts/quickwin-4-recipe-nodejs-postgres.md](.claude/prompts/quickwin-4-recipe-nodejs-postgres.md)

---

### <a name="prompt-5"></a>Quick Win #5: Platform Quick Install

**File:** `.claude/prompts/quickwin-5-platform-quick-install.md`

**Aufgabe:**
- Quick Install Guide für Platform Teams
- Production-ready in 4 Stunden

**Outcome:**
```
docs/platform-team-guide/quick-install.md

Step 1: Setup PostgreSQL (30 min)
Step 2: Configure OIDC (20 min)
Step 3: Install with Helm (15 min)
Step 4: Setup Monitoring (30 min)
Step 5: Configure Alerts (15 min)
Step 6: Onboard First User (10 min)

Total: 4 hours → Production-ready ✓
```

**Success Metric:**
- Platform Team kann in 4h innominatus deployen (statt 2-3 Tage)

**Prompt:** [.claude/prompts/quickwin-5-platform-quick-install.md](.claude/prompts/quickwin-5-platform-quick-install.md)

---

## Execution Plan (1 Week Sprint)

### Day 1 (Monday): Foundation
```
Morning:
- Execute Quick Win #1: README.md Persona Split
- Review & merge

Afternoon:
- Create placeholder directory structure:
  docs/user-guide/
  docs/platform-team-guide/
  docs/development/
```

### Day 2 (Tuesday): User Path
```
Full Day:
- Execute Quick Win #2: User Guide - Getting Started
- Test with mock user journey
```

### Day 3 (Wednesday): CLI Improvement
```
Morning:
- Execute Quick Win #3: CLI Help Separation
- Test all CLI modes

Afternoon:
- Update CLI test suite
- Smoke test
```

### Day 4 (Thursday): Real Examples
```
Full Day:
- Execute Quick Win #4: Recipe Node.js + PostgreSQL
- Create working example code
- Test end-to-end deployment
```

### Day 5 (Friday): Platform Team
```
Morning:
- Execute Quick Win #5: Platform Quick Install

Afternoon:
- Final review all changes
- Update links
- Smoke test all paths
- Create BEFORE/AFTER comparison
```

---

## How to Use These Prompts

### Option 1: Sequential Execution

```bash
# Day 1
claude --prompt .claude/prompts/quickwin-1-readme-persona-split.md

# Day 2
claude --prompt .claude/prompts/quickwin-2-user-guide-getting-started.md

# Day 3
claude --prompt .claude/prompts/quickwin-3-cli-help-separation.md

# Day 4
claude --prompt .claude/prompts/quickwin-4-recipe-nodejs-postgres.md

# Day 5
claude --prompt .claude/prompts/quickwin-5-platform-quick-install.md
```

### Option 2: Parallel Execution (if team of 2)

```bash
# Person A (User-focused)
Day 1: Quick Win #1 (README)
Day 2: Quick Win #2 (User Getting Started)
Day 3: Quick Win #4 (Recipe)

# Person B (Platform-focused)
Day 1: Quick Win #1 (README) - review
Day 2: Quick Win #5 (Platform Install)
Day 3: Quick Win #3 (CLI Help)
```

### Option 3: Interactive Mode

```bash
# Open prompt in editor
code .claude/prompts/quickwin-1-readme-persona-split.md

# Copy prompt to Claude
# Execute
# Review output
# Iterate if needed
```

---

## Success Metrics

### Before (Current State)

| Metric | Value |
|--------|-------|
| README.md length | 870 lines |
| Time to First Success (Developer) | >2 hours |
| Time to Production (Platform Team) | 2-3 days |
| CLI commands shown | 25+ (mixed) |
| Real-world examples | 0 (only hello-world) |
| Persona separation | None |
| Support tickets "How to start?" | ~50/week |

### After (Quick Wins)

| Metric | Value | Improvement |
|--------|-------|-------------|
| README.md length | ~300 lines | -66% |
| Time to First Success (Developer) | <15 min | -92% |
| Time to Production (Platform Team) | 4 hours | -92% |
| CLI commands shown (default) | 9 (user-focused) | -64% |
| Real-world examples | 1 (Node.js+PostgreSQL) | ∞ |
| Persona separation | 3 clear paths | ✓ |
| Support tickets "How to start?" | <10/week | -80% |

### Impact

**Developer Experience:**
- ✅ Sofort klar: "Platform Team hat schon alles aufgesetzt"
- ✅ Erster Deployment in 15 Min statt 2h
- ✅ Copy-paste ready examples
- ✅ CLI ist nicht overwhelming

**Platform Team:**
- ✅ Production-ready in 4h statt 2-3 Tagen
- ✅ Klare Installation Checkliste
- ✅ Monitoring & Alerts out-of-the-box

**Overall:**
- ✅ 80% der DevEx-Probleme gelöst
- ✅ Mit 20% Aufwand (1 Woche)
- ✅ Messbare Verbesserung in allen Metriken

---

## Next Steps (After Quick Wins)

### Week 2-3: Iteration
- Collect user feedback
- Fix issues found
- Add more recipes (Python, Microservices, etc.)
- Improve troubleshooting guides

### Week 4: Documentation Site
- Setup MkDocs or similar
- Convert Markdown to docs site
- Add search
- Host at docs.innominatus.dev

### Month 2: Advanced Topics
- Multi-environment setup
- CI/CD integration
- Custom golden paths
- Security hardening
- Cost optimization

---

## Files Created

This migration plan creates these files:

**Prompts:**
- `.claude/prompts/quickwin-1-readme-persona-split.md`
- `.claude/prompts/quickwin-2-user-guide-getting-started.md`
- `.claude/prompts/quickwin-3-cli-help-separation.md`
- `.claude/prompts/quickwin-4-recipe-nodejs-postgres.md`
- `.claude/prompts/quickwin-5-platform-quick-install.md`

**Documentation:**
- `README.md` (rewritten)
- `docs/user-guide/README.md`
- `docs/user-guide/getting-started.md`
- `docs/user-guide/recipes/nodejs-postgres.md`
- `docs/platform-team-guide/README.md`
- `docs/platform-team-guide/quick-install.md`

**Examples:**
- `examples/nodejs-postgres/score.yaml`
- `examples/nodejs-postgres/app/*`
- `examples/nodejs-postgres/README.md`

**Code Changes:**
- `cmd/cli/main.go` (CLI help separation)
- `internal/cli/client.go` (new commands: tutorial, examples, docs, support)

---

## Checklist

**Before Starting:**
- [ ] Read all 5 prompts
- [ ] Understand the problem (persona mixing)
- [ ] Review current README.md
- [ ] Set aside 1 week for focused work

**During Execution:**
- [ ] Day 1: Quick Win #1 (README)
- [ ] Day 2: Quick Win #2 (User Guide)
- [ ] Day 3: Quick Win #3 (CLI Help)
- [ ] Day 4: Quick Win #4 (Recipe)
- [ ] Day 5: Quick Win #5 (Platform Install)

**After Completion:**
- [ ] Smoke test all paths
- [ ] Update CHANGELOG.md
- [ ] Create BEFORE/AFTER comparison
- [ ] Announce to team
- [ ] Collect feedback

---

## Support

**Questions?**
- Review: `DEVEX_ANALYSIS-DOCUMENTATION_STRUCTURE-2025-10-05.md`
- Review: `DEVEX_ANALYSIS-2025-10-05.md`

**Need help with prompts?**
- Each prompt is self-contained
- Includes context, requirements, acceptance criteria
- Can be executed independently

---

**Created:** 2025-10-05
**Status:** Ready to execute
**Estimated completion:** 1 week
**Expected impact:** 80% of DevEx problems solved

🚀 **Let's fix the documentation!**
