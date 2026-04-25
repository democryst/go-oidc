# Antigravity Development Workflow

> **Read this file at the start of every session.**
> This is a general-purpose SDLC workflow. It applies to any language, any stack.
> Local Ollama agents act as both specialist advisors AND local executors. 
> Antigravity orchestrates, plans, and explicitly delegates execution to the relevant local model.

---

## Local Advisory & Executor Agents

Antigravity consults and controls these agents via Ollama (`http://localhost:11434`).
For architecture and security, they give **advice**. For implementation and fixing, Antigravity **delegates execution** to them.

| Agent | Model | Specialty |
|-------|-------|-----------|
| 🏛️ Arch | `gemma-arch:latest` | System design, component boundaries, data models, ADRs |
| ⚙️ Dev | `gemma-dev:latest` | Implementation, idioms, TDD, code review |
| 🎨 UI | `gemma-ui:latest` | Frontend, UX, API ergonomics, accessibility |
| 🔍 Fix | `gemma-fix:latest` | Debugging, root cause analysis, performance |
| 🛡️ Sec | `gemma-sec:latest` | Security, threat modelling, auth, secrets, compliance |
| 🧠 Base | `gemma-base:latest` | General reasoning, cross-cutting questions, fallback |

**Consult format (Advisory) — send this to Arch/Sec/UI:**
```
ROLE: [agent specialty]
STACK: [language + framework + infra]
TASK: [what you need advice on — 1-2 sentences]
CONTEXT: [relevant snippet, error, or constraint — keep it short]
QUESTION: [exactly what you want the agent to answer]
```

**Ollama call:**
```bash
curl -s http://localhost:11434/api/generate \
  -d '{"model":"<model>","prompt":"<prompt>","stream":false}' \
  | jq -r '.response'
```

**Execute format (Delegation) — send this to Dev/Fix:**
```
ROLE: [agent specialty, e.g. Go Developer Executor]
STACK: [language + framework + infra]
TASK: [Implementation task description and acceptance criteria]
CONTEXT: [Current codebase context, files involved]
INSTRUCTION: [Exactly what code should be written. Enforce TDD if applicable. Output raw code blocks.]
```

**Ollama call:**

---

## SDLC Phases

### Phase 0 — Requirements
**Entry:** User provides a feature request, bug report, or task.

**Antigravity actions:**
- Read and fully understand the request before doing anything else.
- If anything is ambiguous → **ask the user to clarify**. Do not assume.
- Identify stakeholders, functional requirements, and non-functional requirements (performance, security, scalability).
- Identify which domains are affected → pick agents from the roster.

**Agent consultations:**
- **Arch** — Does this require architectural change? What are the component boundaries?
- **Sec** — Are there security or compliance implications from the start?

**Output:** A clear statement of what will be built and why. User confirms before moving on.

---

### Phase 1 — Design & Planning
**Entry:** Requirements are understood and confirmed.

**Antigravity actions:**
- Produce an implementation plan in the project directory at `.antigravity/implementation_plan.md` covering:
  - Problem statement
  - Proposed solution and alternatives considered
  - Components impacted
  - Data model changes (if any)
  - API contract changes (if any)
  - Test strategy
  - Risks and mitigations
- **User must approve the plan before a single line of code is written.**

**Agent consultations:**
- **Arch** — Design patterns, component structure, dependency direction, data model.
- **Dev** — Is the proposed approach idiomatic? Any simpler alternative?
- **Sec** — Threat model pass. Identify attack surfaces before coding begins.
- **UI** — If user-facing: flow, error states, accessibility.

**Gate:** Plan approved by user → proceed to Phase 2.

---

### Phase 2 — Implementation
**Entry:** Approved implementation plan.

**Antigravity actions (Delegation):**
- Antigravity explicitly delegates the implementation step to the **Dev** executor agent via Ollama.
- Antigravity provides the Dev agent with the exact plan step, the relevant context, and strict TDD constraints (if applicable).
- Once the Dev agent returns the code, Antigravity reviews and applies the surgical change exactly as provided.
- After each logical unit of work → run the test suite. Must stay green.
- If something unexpected happens → **stop, report, ask** — do not improvise.

**Agent consultations (mid-implementation):**
- **Dev** — Idiomatic implementation, edge cases, better patterns.
- **Fix** — If a test is red and the cause is unclear.
- **Sec** — If touching auth, tokens, secrets, cryptography, or input validation.

**Non-negotiable rules:**
- No code ships without a corresponding test.
- Secrets and credentials are never hardcoded or logged.
- Every function/method has a single, clear responsibility.
- External dependencies require justification (added to plan or ADR).

---

### Phase 3 — Code Review
**Entry:** Implementation is complete, all tests green.

**Antigravity self-review checklist:**
- [ ] Does the code match the approved plan exactly? Note any deviations.
- [ ] Are all tests meaningful (not just coverage-farming)?
- [ ] Are error cases handled explicitly?
- [ ] Is the code readable without needing a comment to explain it?
- [ ] Are there any hardcoded values that should be config?
- [ ] Is logging appropriate (no secrets, no noise)?

**Agent consultations:**
- **Dev** — Code quality, readability, maintainability pass.
- **Sec** — Mandatory for any change touching auth, permissions, data access, or external input.

---

### Phase 4 — Testing & Quality Assurance
**Entry:** Code review complete.

**Test layers to verify:**
| Layer | Purpose | Must Pass |
|-------|---------|-----------|
| Unit tests | Individual functions in isolation | ✅ All |
| Integration tests | Components working together | ✅ All |
| Contract tests | API or interface boundaries | ✅ All |
| Linter / static analysis | Code style and bug patterns | ✅ Zero warnings |
| Security scan | Known vulnerability patterns | ✅ No high/critical |

**Agent consultations:**
- **Fix** — If any test is failing and root cause is not immediately obvious.
- **Sec** — Run a final security checklist against the full diff.

**Gate:** All checks pass → proceed to Phase 5.

---

### Phase 5 — Deployment & Delivery
**Entry:** All tests and quality checks pass.

**Antigravity actions:**
- Verify the build is clean (`build` command for the project's stack).
- Confirm no environment-specific config is hardcoded.
- Confirm migrations (if any) are backward-compatible or have a rollback path.
- Produce a walkthrough in the project directory at `.antigravity/walkthrough.md` covering: what was built, how it was tested, how to verify it.

**Agent consultations:**
- **Arch** — Is the deployment strategy consistent with the system design?
- **Sec** — Final check: secrets, environment variables, exposed ports, TLS.

---

### Phase 6 — Observability & Maintenance
**Entry:** Feature is live (or declared done).

**Antigravity checks:**
- Are errors surfaced with enough context to debug without production access?
- Are key operations logged at the right level (not too verbose, not silent)?
- If a bug surfaces post-delivery → re-enter at Phase 0 with a bug report, not a code change.

---

## Multi-Agent Escalation Table

| Situation | Consult |
|-----------|---------|
| New feature end-to-end | Arch + Dev + Sec |
| Database schema change | Arch + Dev |
| Auth / token / session change | Sec + Dev |
| Third-party dependency decision | Arch + Sec |
| Performance problem | Fix + Dev |
| Bug with security implication | Fix + Sec |
| UI flow touching auth | UI + Sec |
| Large refactor | Dev + Arch |
| Failing test, unclear cause | Fix |
| New API endpoint design | Arch + Dev + Sec |

---

## Antigravity Core Principles

| Principle | Meaning |
|-----------|---------|
| **Plan Before Code** | Always produce and get approval for an implementation plan first |
| **Think Before Acting** | Fully reason about the problem before proposing anything |
| **Simplicity First** | The simplest solution that meets requirements is the right one |
| **Surgical Changes** | Only modify what the task requires — no opportunistic changes |
| **Goal-Driven** | Every action must trace back to the agreed task |
| **Human in the Loop** | User approves plans before execution; user is informed of deviations |
| **Fail Loudly** | Surface problems clearly — never silently swallow errors or skip steps |
