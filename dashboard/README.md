# ADM Battle Console (dashboard)

A realtime web dashboard for the Agentic Defense Matrix red/blue/green exercise.
It shows, live, how the defenses are holding: service health, a battle scoreboard
(attacks, block rate, detection rate, remediations, MTTR), **successful attack
chains** (adaptive LLM follow-ups + green SOC summaries), a per-technique
blocked-vs-landed breakdown, a streaming event feed, and recent attack→remediation
sessions.

**Live:** https://jest-test-team.github.io/Agentic-Defense-Matrix-ADM/

## What it talks to

The dashboard is a **static site** (Next.js `output: "export"`) — no server. In the
browser it:

- polls the analysis API `GET /api/stats`, `/api/timeline`, and `/api/chains` every few seconds,
- opens a Server-Sent-Events stream `GET /api/stream` for live events,
- probes `/health`, `/ready` (Neon), and the gateway `/v1/health`,
- opens chain detail via `GET /api/chains/:id` (steps, strategy, remediation summary).

Because GitHub Pages is HTTPS, the API must also be HTTPS (see the Caddy setup in
`docs/architecture/live-deployment.md`). The endpoint is **runtime-configurable**
so the same build can point at any deployment:

- `?api=https://host&gw=https://host` in the URL, or
- the **Endpoint** box at the bottom of the page (persisted to `localStorage`).

The default is `https://api.dennisleehappy.org` (Caddy fronts both APIs; it routes
`/v1/*` to the gateway and everything else to the analysis engine, so one host
serves both).

## Internationalization

English and Traditional Chinese (繁體中文), toggled in the header and persisted;
first visit auto-detects `zh-*` browsers. All strings live in `lib/i18n.ts` — add a
language by extending the `translations` map. The collapsible **About** panel
explains the project's purpose and the three teams in both languages.

## Layout

| File | Purpose |
|------|---------|
| `app/page.tsx` | Main battle console (scoreboard, feed, chains, matrix preview) |
| `app/matrix/page.tsx` | Full 10k-variant corpus browser |
| `app/search/page.tsx` | Elasticsearch full-text search |
| `lib/api.ts` | Typed client for analysis/gateway endpoints |
| `lib/i18n.ts` | EN / 繁中 strings |
