# ADR-007: Intent-Drift Detection as a Pluggable, Streaming Model

## Status

Accepted (research direction; see [docs/research/](../research/)).

## Context

The shipped L7 detector (`pkg/semantic/analyzer.go`) is lexical keyword matching +
frequency analysis over a sliding window. That is fine engineering but weak science
and — as measured — **blind to obfuscation**: any mutation that breaks the literal
keyword (base64, hex, zero-width, homoglyph, paraphrase) drives its signal to zero.
To make ADM a defensible research contribution we needed (a) a formal model that
*generalizes* the keyword detector, and (b) evidence that a better feature map wins.

## Decision

Model detection as **intent drift**: score the *trajectory* `x_{1:t}`, not i.i.d.
messages. Define a pluggable feature map φ and a windowed statistic
`D_t(W) = mean over W of d_M(φ(a_i), C)` firing at θ, where `C` is the authorization
manifold. The lexical detector is the **degenerate instance** (φ = keyword
indicators). Ship two φ behind one interface (`pkg/semantic/featurizer.go`,
`drift.go`):

- `LexicalScorer` — keyword-mass drift (faithful to production).
- `ManifoldScorer` — dense char-n-gram embedding, cosine distance to the manifold.

The `DriftDetector` is `O(1)` amortized per token (ring-buffer window sum) — the
basis of the **asymmetry principle** `α = o(1)` vs a per-message LLM guard.

## Consequences

- The keyword-vs-embedding **ablation** is a config swap, isolating φ quality from
  the pipeline (embedding 51.5% vs lexical 13.8% detection at matched FPR; +100 pts
  on obfuscation). FP/FN bounds (Eq. 2/3) are φ-agnostic and empirically confirmed by
  the window-W sweep. See `docs/research/{ablation,sweep}-results.md`.
- The production `analyzer.go` is unchanged (backward-compatible); the drift detector
  is additive and gated for future adoption.
- **Next:** swap the hashing embedding for a learned sentence embedding behind the
  same interface (expected to widen the margin), and add a Llama Guard SOTA arm for
  the measured asymmetry (`cmd/baseline`).

## Alternatives Considered

| Alternative | Rejected Because |
|---|---|
| Keep keyword-only detection | blind to obfuscation; not a research contribution |
| Per-message LLM classifier (Llama Guard) as the detector | `α = Θ(1)` — loses the throughput race; no cross-message accumulation |
| Replace `analyzer.go` outright | breaks production; the manifold model *subsumes* it as a special case instead |
