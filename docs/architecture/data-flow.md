# Data Flow Architecture

> Related: [battle-orchestration.md](../battle-orchestration.md),
> [live-deployment.md](live-deployment.md), [ADR-006](../adr/006-hosted-llm-failover.md),
> [ADR-008](../adr/008-llm-red-green-teams.md).

## Event Ingestion Pipeline (blue-team SIEM path)

```mermaid
flowchart LR
    subgraph "Event Sources"
        GW[Gateway]
        WD[Watchdog]
        AG[Agents]
    end

    subgraph "Ingestion"
        SIEM[SIEM Engine]
        RB[Ring Buffer<br/>64K events]
    end

    subgraph "Storage Tiers"
        REDIS[(Redis Streams<br/>Hot: 7d)]
        WARM[Warm Storage<br/>JSONL: 30d]
        COLD[(Cold Storage<br/>S3: 180d)]
    end

    subgraph "Processing"
        CORR[Correlation Engine]
        RULES[Rule Evaluator]
        ALERT[Alert Dispatcher]
    end

    subgraph "Response"
        WEBHOOK[Webhook → Gateway]
        KILL[Kill Container]
        BLOCK[Block Egress]
    end

    GW -->|OTLP| SIEM
    WD -->|Syscall Events| SIEM
    AG -->|Tool Results| SIEM
    SIEM --> RB
    RB --> REDIS
    REDIS -->|7d retention| WARM
    WARM -->|30d retention| COLD
    REDIS --> CORR
    CORR --> RULES
    RULES -->|Threshold met| ALERT
    ALERT --> WEBHOOK
    ALERT --> KILL
    ALERT --> BLOCK
```

## Semantic Analysis Pipeline (blue-team boundary)

Local keyword / similarity scoring at the gateway — **not** hosted LLM.

```mermaid
flowchart TD
    A[User Prompt] --> B[Tokenize]
    B --> C[Embed<br/>Local Model]
    C --> D[Similarity Check<br/>vs Known Patterns]
    D --> E{Score > Threshold?}
    E -->|Clean| F[Pass to OPA]
    E -->|Suspicious| G[Rate Limit]
    E -->|Malicious| H[Block + Alert SIEM]
    F --> I[OPA Policy Check]
    I -->|Allowed| J[Route to Agent]
    I -->|Denied| K[Block + Log]
```

## Agent Execution Pipeline (blue-team agents)

Planner / summarizer call the hosted LLM (Groq → X.AI) or on-box Ollama via
`pkg/ollama.NewClientFromEnv()`.

```mermaid
flowchart TD
    A[Plan Request] --> B[Planner Agent]
    B --> C[LLM: Generate Steps]
    C --> D[Return PlannedStep[]]

    D --> E{For Each Step}
    E --> F[Executor Agent]
    F --> G[Create Ephemeral Container]
    G --> H[Mount Tool Schema]
    H --> I[Execute Tool Call]
    I --> J[Watchdog: Monitor Syscalls]
    J --> K{Anomaly?}
    K -->|No| L[Collect Result]
    K -->|Yes| M[Kill + Alert]
    L --> N[Report to Gateway]

    N --> O[Summarizer Agent]
    O --> P[LLM: Generate Summary]
    P --> Q[Return Response]
```

## Battle end-to-end (red → blue → green → analysis)

```mermaid
sequenceDiagram
    participant RT as RedAgent
    participant LLM as Groq_XAI
    participant GW as Gateway
    participant Sem as Semantic
    participant Ana as Analysis_DB
    participant Redis as Redis_adm_battle
    participant GT as GreenAgent
    participant Dash as Dashboard

    RT->>RT: GenerateCorpus or LLM next step
    RT->>GW: POST chat or tools with X-Session-ID and chain_id
    GW->>Sem: Analyze prompt
    alt malicious
        GW-->>RT: 403 blocked
        RT->>Ana: emit attack blocked
        RT->>Redis: XADD
    else allowed through boundary
        GW->>LLM: Chat OpenAI-compat
        LLM-->>GW: completion
        GW-->>RT: 2xx allowed
        RT->>Ana: emit attack allowed plus chain upsert
        RT->>Redis: XADD
        RT->>LLM: AdaptiveMutate next payload and strategy
        LLM-->>RT: next step
        Redis->>GT: allowed attack
        GT->>LLM: TriageRemediation
        LLM-->>GT: revoke restart_targets summary
        GT->>GW: revoke if decided
        GT->>GT: restart chosen agents
        GT->>Ana: emit remediation with summary
        Dash->>Ana: GET api chains
    end
```

## Who calls hosted LLM

| Caller | When | Purpose |
|--------|------|---------|
| Gateway / planner / summarizer | Allowed chat / plan / summarize | Target-system inference |
| `redteam_agent` | Only on `outcome=allowed` (landing) | Adaptive mutation + next-technique strategy |
| `greenteam_agent` | On landed-attack remediation | Severity triage, revoke/restart decisions, SOC summary |

Day-to-day corpus attacks are **deterministic** (no LLM). LLM failure falls back
to deterministic mutation / “always revoke + restart target”. Flags:
`ADM_RED_LLM`, `ADM_GREEN_LLM`. Backend: same Groq → X.AI client as ADR-006.

## Attack chain persistence

Successful multi-step attacks share a `chain_id` in battle-event `labels`.

- Tables: `attack_chains`, `attack_chain_steps` (see `analysis/migrations/002_attack_chains.sql`)
- Ingest upserts the chain when `labels.chain_id` is present
- Dashboard: `GET /api/chains?status=landed`, `GET /api/chains/:id`

Label conventions:

| key | Writer | Meaning |
|-----|--------|---------|
| `chain_id` | red | Attack-chain UUID |
| `chain_step` | red | Step index |
| `mutation_source` | red | `deterministic` \| `llm_adaptive` |
| `strategy` | red | Short strategy phrase |
| `summary` | green | SOC remediation narrative |
| `triage` | green | revoke / restart decision summary |

## Green Team Auto-Response (battle path)

Green team watches Redis `adm:battle` for red `allowed` attacks, optionally
asks the hosted LLM for triage, then remediates.

```mermaid
sequenceDiagram
    participant Redis as Redis adm:battle
    participant GT as Green team
    participant LLM as Groq or X.AI
    participant GW as Gateway
    participant Docker as Docker API
    participant Ana as Analysis

    Redis->>GT: attack outcome allowed
    GT->>LLM: TriageRemediation
    LLM-->>GT: severity revoke targets summary
    alt revoke true
        GT->>GW: POST admin revoke session
    end
    loop each allowed restart target
        GT->>Docker: restart adm.role=agent container
    end
    GT->>Ana: remediation event with summary
```
