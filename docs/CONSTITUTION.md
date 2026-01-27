# Project Constitution

**Project Name:** MagnetPlay  
**Version:** 1.0  
**Effective Date:** January 27, 2026  
**Status:** Active

---

## Table of Contents

- [1. Mission Statement](#1-mission-statement)
- [2. Core Values](#2-core-values)
- [3. Architectural Principles](#3-architectural-principles)
- [4. Decision-Making Framework](#4-decision-making-framework)
- [5. Development Standards](#5-development-standards)
- [6. Quality Gates](#6-quality-gates)
- [7. Team Structure & Responsibilities](#7-team-structure--responsibilities)
- [8. Communication Protocols](#8-communication-protocols)
- [9. Change Management](#9-change-management)
- [10. Conflict Resolution](#10-conflict-resolution)

---

## 1. Mission Statement

To build a high-performance, peer-to-peer video streaming system that enables immediate playback of torrent content with exceptional user experience, while maintaining code quality, system reliability, and team sustainability.

### 1.1 Success Criteria

- **User Experience:** First frame rendered within 10 seconds of torrent selection
- **Performance:** Achieve 85% cache hit rate for optimal streaming
- **Reliability:** 99.5% uptime for backend services
- **Code Quality:** Maintain 80%+ test coverage across all components
- **Team Health:** Sustainable development pace with minimal technical debt

---

## 2. Core Values

### 2.1 Engineering Excellence

We are committed to:
- Writing clean, maintainable, and well-documented code
- Comprehensive testing at all levels (unit, integration, e2e)
- Continuous learning and improvement
- Code reviews as opportunities for knowledge sharing

### 2.2 User-Centric Design

We prioritize:
- Intuitive user interfaces with minimal friction
- Responsive design across all devices
- Graceful error handling and informative feedback
- Accessibility for all users

### 2.3 Transparency

We believe in:
- Open communication within the team
- Clear documentation of decisions and trade-offs
- Visible progress through metrics and dashboards
- Honest assessment of challenges and blockers

### 2.4 Collaboration

We foster:
- Cross-functional team involvement in design decisions
- Pair programming for complex features
- Regular knowledge-sharing sessions
- Supportive code review culture

### 2.5 Sustainability

We maintain:
- Reasonable working hours and work-life balance
- Manageable sprint commitments
- Technical debt allocation (20% of capacity)
- Continuous refactoring as part of development

---

## 3. Architectural Principles

### 3.1 Separation of Concerns

**Principle:** Each component has a single, well-defined responsibility.

**Implementation:**
- Frontend handles UI/UX and user interactions only
- Backend manages HTTP streaming and orchestration
- Sidecar owns torrent protocol and piece management

**Why:** Enables independent scaling, testing, and deployment of components.

### 3.2 API-First Design

**Principle:** Define APIs before implementation.

**Implementation:**
- Document REST and gRPC contracts in OpenAPI/Protobuf
- Generate client/server stubs from specs
- Version all APIs (v1, v2) with deprecation policies

**Why:** Ensures contract stability and enables parallel development.

### 3.3 Fail-Fast & Resilient

**Principle:** Detect failures early and handle gracefully.

**Implementation:**
- Input validation at API boundaries
- Circuit breakers for external dependencies
- Timeout policies on all network calls
- Comprehensive error logging

**Why:** Prevents cascading failures and improves debuggability.

### 3.4 Observable by Default

**Principle:** Every component exposes health and metrics.

**Implementation:**
- Structured logging (JSON) in all services
- Prometheus metrics endpoints
- Distributed tracing with correlation IDs
- Health check endpoints

**Why:** Enables proactive monitoring and rapid troubleshooting.

### 3.5 Data Privacy & Security

**Principle:** Handle user data responsibly and securely.

**Implementation:**
- No persistent storage of user identities
- HTTPS for all API communications
- Input sanitization to prevent injection attacks
- Rate limiting on public endpoints

**Why:** Builds trust and complies with privacy standards.

---

## 4. Decision-Making Framework

### 4.1 Decision Categories

| Category | Decision Maker | Approval Required | Examples |
|----------|---------------|-------------------|----------|
| **Strategic** | Tech Lead + Product Owner | Team Consensus | Architecture changes, tech stack |
| **Tactical** | Team Lead | Code Review | Implementation approach, design patterns |
| **Operational** | Individual Developer | None | Variable names, minor refactors |

### 4.2 Consensus Building

For strategic decisions:
1. **Proposal Phase** (2 days): Author writes RFC with context, options, and recommendation
2. **Discussion Phase** (3 days): Team reviews and provides feedback
3. **Decision Phase** (1 day): Tech Lead synthesizes input and makes final call
4. **Documentation Phase** (1 day): Decision recorded in `docs/decisions/`

### 4.3 Reversibility

Decisions are categorized as:
- **Type 1 (One-way doors):** Hard to reverse (e.g., database choice) → Require consensus
- **Type 2 (Two-way doors):** Easy to reverse (e.g., UI layout) → Fast decision by team lead

### 4.4 Escalation Path

If consensus cannot be reached:
1. Tech Lead makes a time-boxed decision (7 days max)
2. Team commits to decision for one sprint
3. Retrospective to evaluate and potentially reverse

---

## 5. Development Standards

### 5.1 Code Style

**Language-Specific Guides:**
- **Java:** Google Java Style Guide + Spring Boot best practices
- **Go:** Effective Go + Go Code Review Comments
- **TypeScript/React:** Airbnb React/JSX Style Guide
- **Markdown:** GitHub-flavored Markdown for all documentation

**Automation:**
- Pre-commit hooks enforce formatting (Prettier, gofmt, google-java-format)
- CI pipeline fails on style violations

### 5.2 Branching Strategy

**Model:** GitHub Flow (simplified Git Flow)

```
main (protected)
  ├── feature/add-subtitle-support
  ├── bugfix/fix-seek-crash
  └── hotfix/critical-memory-leak
```

**Rules:**
- `main` is always deployable
- Feature branches created from `main`
- Pull requests require 2 approvals
- Squash merge to keep history clean

### 5.3 Commit Messages

**Format:** Conventional Commits

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Example:**
```
Feature(streaming): add support for HTTP 206 range requests

Implement byte-range serving in StreamingController to enable
video seeking. Includes logic for piece availability checking
and 416 Range Not Satisfiable error handling.

Closes #142
```

**Types:** `Feature`, `Bugfix`, `Docs`, `Style`, `Refactor`, `Test`, `Chore`

### 5.4 Documentation Requirements

Every pull request must include:
- **Code Comments:** Public APIs, complex logic
- **README Updates:** If new component or configuration added
- **API Documentation:** If endpoints changed
- **Changelog Entry:** If user-facing change

---

## 6. Quality Gates

### 6.1 Code Review Checklist

Before approving a PR, reviewers must verify:
- [ ] Code follows style guide
- [ ] Tests added for new functionality
- [ ] No commented-out code or debug statements
- [ ] Documentation updated
- [ ] No hardcoded credentials or secrets
- [ ] Error handling is comprehensive
- [ ] Performance considerations addressed

### 6.2 Automated Checks (CI)

All PRs must pass:
- [ ] Unit tests (80%+ coverage)
- [ ] Linting and formatting
- [ ] Static analysis (SonarQube)
- [ ] Build succeeds
- [ ] Integration tests pass
- [ ] Security scan (OWASP dependency check)

### 6.3 Definition of Done

A task is considered complete when:
- [ ] Code merged to `main`
- [ ] Tests written and passing
- [ ] Reviewed by 2+ team members
- [ ] Documentation updated
- [ ] Deployed to staging
- [ ] Acceptance criteria validated
- [ ] No critical bugs identified

### 6.4 Performance Benchmarks

Before merging performance-critical changes:
- [ ] Benchmark tests included
- [ ] No regression > 10% in latency/throughput
- [ ] Memory usage profiled and acceptable
- [ ] Load test results documented

---

## 7. Team Structure & Responsibilities

### 7.1 Team Composition

**Team Alpha (Frontend & Backend)**
- **Lead:** Senior Full-Stack Developer
- **Members:** 2 Frontend, 1 Backend Developer
- **Focus:** User interface, REST APIs, WebSocket streaming

**Team Beta (Sidecar & Infrastructure)**
- **Lead:** Senior Backend/DevOps Developer
- **Members:** 2 Backend, 1 DevOps Engineer
- **Focus:** Torrent engine, gRPC services, deployment

**Shared Responsibilities:**
- **Tech Lead:** Cross-team architecture and technical direction
- **Product Owner:** Prioritization and roadmap
- **QA Engineer:** Test automation and quality assurance

### 7.2 Roles & Responsibilities

| Role | Responsibilities |
|------|------------------|
| **Tech Lead** | Architecture design, technical debt management, tech radar |
| **Team Lead** | Sprint planning, impediment removal, team health |
| **Developer** | Feature implementation, code review, testing |
| **DevOps Engineer** | CI/CD, monitoring, infrastructure as code |
| **QA Engineer** | Test strategy, automation, bug triage |
| **Product Owner** | Backlog prioritization, stakeholder communication |

### 7.3 Rotation Policies

To prevent knowledge silos:
- **Oncall Rotation:** Weekly rotation for production support
- **Code Review Rotation:** Automated assignment to distribute expertise
- **Pair Programming:** Encouraged across team boundaries
- **Documentation Days:** Last Friday of month for documentation

---

## 8. Communication Protocols

### 8.1 Daily Standup

**Format:** Async via Slack
- **When:** Before 10 AM local time
- **Template:**
  ```
  ✅ Yesterday: Completed X, Y
  🎯 Today: Working on Z
  🚧 Blockers: None / Need help with A
  ```

### 8.2 Sprint Ceremonies

| Ceremony | Duration | Participants | Purpose |
|----------|----------|--------------|---------|
| **Sprint Planning** | 2 hours | All team | Select and estimate work |
| **Daily Standup** | 15 min | All team | Synchronize progress |
| **Sprint Review** | 1 hour | All team + stakeholders | Demo completed work |
| **Retrospective** | 1 hour | All team | Continuous improvement |
| **Backlog Refinement** | 1 hour | Team leads + PO | Prepare upcoming work |

### 8.3 Communication Channels

- **Slack #engineering:** General technical discussions
- **Slack #deployments:** Release announcements
- **Slack #incidents:** Production issues
- **GitHub Discussions:** Design proposals (RFCs)
- **Confluence:** Long-form documentation
- **Email:** External communication only

### 8.4 Response Time Expectations

| Priority | Channel | Response Time |
|----------|---------|--------------|
| **P0 (Production Down)** | Slack #incidents + Phone | 15 minutes |
| **P1 (Critical Bug)** | Slack #engineering | 2 hours |
| **P2 (Code Review)** | GitHub | 24 hours |
| **P3 (General Question)** | Slack | 48 hours |

---

## 9. Change Management

### 9.1 Technology Changes

**Adding New Technology:**
1. Write RFC explaining need, alternatives, and risks
2. Build proof-of-concept (max 2 days)
3. Present findings to team
4. Get consensus approval
5. Update tech radar

**Examples:**
- Switching from REST to GraphQL
- Adding new monitoring tool
- Adopting new framework

### 9.2 Process Changes

**Modifying Development Process:**
1. Propose change in retrospective
2. Trial for one sprint
3. Collect feedback
4. Adopt, adapt, or reject

**Examples:**
- Changing sprint length
- Modifying code review process
- Adjusting meeting schedules

### 9.3 Constitution Amendments

**Updating This Document:**
1. Any team member can propose amendment
2. Requires 75% team approval
3. Merge to `main` updates constitution
4. Announce to all stakeholders

**Versioning:**
- Major changes (v1.0 → v2.0): Structural changes
- Minor changes (v1.0 → v1.1): Additions or clarifications
- Patch changes (v1.0.0 → v1.0.1): Typo fixes

---

## 10. Conflict Resolution

### 10.1 Technical Disagreements

**Level 1: Peer Discussion**
- Two developers disagree on implementation
- Action: 30-minute video call to discuss trade-offs
- Outcome: Reach consensus or escalate

**Level 2: Team Lead Mediation**
- Discussion doesn't resolve after 1 day
- Action: Team lead facilitates decision
- Outcome: Binding decision for current sprint

**Level 3: Tech Lead Arbitration**
- Decision impacts multiple teams
- Action: Tech lead makes final call
- Outcome: Documented in decision log

### 10.2 Interpersonal Conflicts

**Process:**
1. **Private Conversation:** Parties discuss directly
2. **Team Lead Mediation:** If unresolved after 2 days
3. **HR Involvement:** If policy violation suspected
4. **External Mediation:** Last resort

**Principles:**
- Assume positive intent
- Focus on behavior, not personality
- Keep discussions private
- Document outcomes, not details

### 10.3 Prioritization Conflicts

**Scenario:** Team wants to refactor, PM wants features

**Resolution:**
1. **Quantify Impact:** Estimate time saved by refactor
2. **Negotiate Allocation:** Agree on % split (e.g., 70% features, 30% tech debt)
3. **Trial Period:** Re-evaluate after one sprint
4. **Escalate:** If no agreement, Product Owner and Tech Lead decide

---

## Appendix A: Glossary

| Term | Definition |
|------|------------|
| **RFC** | Request for Comments - design proposal document |
| **Tech Radar** | Inventory of technologies with adopt/trial/assess/hold status |
| **Tech Debt** | Suboptimal code requiring future refactoring |
| **Oncall** | Designated developer responsible for production issues |
| **P0/P1/P2/P3** | Priority levels (0=critical, 3=low) |

---

## Appendix B: Document History

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 1.0 | 2026-01-27 | Initial constitution | Architecture Team |

---

## Signatures

By contributing to this project, all team members agree to abide by this constitution.

**Tech Lead:** ___________________________  
**Team Alpha Lead:** ___________________________  
**Team Beta Lead:** ___________________________  
**Product Owner:** ___________________________

---

**Last Updated:** January 27, 2026  
**Next Review:** April 27, 2026 (Quarterly)  
**Status:** Active
