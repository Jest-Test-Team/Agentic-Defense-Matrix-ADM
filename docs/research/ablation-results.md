# Ablation Results — embedding-φ vs lexical-φ

**Runnable experiment** for contribution C1
([formalization-intent-drift.md](formalization-intent-drift.md)). It holds the
windowed drift detector `D_t(W)` fixed and swaps the drift scorer φ, isolating the
*embedding gain* (φ quality) and the *windowing gain* (variance reduction) as two
separate, measured effects.

```bash
go run ./cmd/ablation                 # human table
go run ./cmd/ablation -json > r.json  # machine-readable (docs/research/ablation-results.json)
go test ./pkg/semantic/...            # the pointwise claims, CI-protected
```

Deterministic: attack corpus is `GenerateCorpus(4000, seed=1337)`; benign eval set
and manifold are fixed; embedding is char-3-gram feature hashing (dim 256, no
learned weights), so results reproduce byte-for-byte.

## Setup

- **φ arms.** `lexical` = keyword-mass drift, faithful to the shipped
  `pkg/semantic/analyzer.go`; `embedding` = `1 − max cosine` to the authorization
  manifold under a dense char-n-gram φ.
- **Manifold.** System prompt + authorized tool descriptions + benign in-scope
  examples (`semantic.AuthorizedCorpus`).
- **Calibration.** θ is calibrated *per arm* to a matched **benign FPR ≈ 2%** — the
  comparison is at equal false-positive cost, not equal θ.

## Result 1 — embedding gain (single-step detection, matched 2% FPR)

| φ | θ | benign FPR | overall detect |
|---|---|---|---|
| lexical (shipped) | 0.000 | 0.0% | **13.8%** |
| embedding | 0.692 | 2.0% | **51.5%** |

**Detection by mutation class** — the story is in the obfuscations:

| mutation | lexical | embedding | Δ (embed − lex) |
|---|--:|--:|--:|
| identity | 24.6% | 32.4% | +7.8 |
| case-flip | 26.8% | 33.9% | +7.0 |
| nesting | 24.3% | 31.8% | +7.5 |
| multilang-zh | 23.9% | 27.9% | +4.0 |
| whitespace-pad | 9.9% | 36.0% | +26.0 |
| url-encode | 8.1% | 59.2% | +51.2 |
| hex | 0.0% | 50.5% | +50.5 |
| base64 | 0.0% | 63.9% | +63.9 |
| homoglyph | 2.7% | 81.2% | **+78.5** |
| concat-split | 0.0% | 97.8% | **+97.8** |
| zero-width | 0.0% | 100.0% | **+100.0** |

**Reading.** Wherever a mutation destroys the literal keyword — `base64`, `hex`,
`zero-width`, `concat-split`, `homoglyph` — the lexical scorer collapses to **0%**
because the keyword mass is empty. The embedding scorer, representing sub-word
*shape*, still places the payload far off the authorization manifold and detects it.
This is the quantitative case that the shipped keyword detector is a **degenerate
instance** whose blind spot is exactly the corpus's obfuscation classes.

## Result 2 — windowing gain (sustained camouflage, matched FPR)

The windowed mean cannot beat per-step detection *at the same θ* (mean ≤ max). The
gain is **variance reduction**: averaging shrinks the benign statistic's spread
(Eq. 2, FPR decays ∝ exp(−W·margin²)), so the windowed detector runs a **lower**
θ_W at the same benign FPR and catches sustained-moderate drift a per-step detector
must let through.

| detector | θ | detect (sustained camouflage) |
|---|---|--:|
| per-step, W=1 | θ₁ = 0.692 | 22.1% |
| windowed, W=6 | θ_W = 0.653 | **31.9%** (+9.7 pts) |

θ_W < θ₁ is the mechanism; the +9.7 pts is the payoff, at identical false-positive cost.

## What this substantiates for the paper

- The **embedding-φ ablation** (Result 1) isolates φ quality from the pipeline and
  shows a large, honest gain concentrated exactly on obfuscation — reviewer objection
  R1 ("it's just keyword matching") is answered with a swap-in-place experiment.
- The **windowing ablation** (Result 2) empirically exhibits the variance-reduction
  mechanism behind the FP/FN bounds (Eq. 2/3) and the detection–latency law.

## Next (to reach a full figure set)

- Sweep `W` and overlay measured FPR/FNR on the Eq. 2/3 exponential prediction
  (the theory-matches-experiment plot).
- Add a **Llama Guard** arm (per-message LLM) for the SOTA baseline + the asymmetry
  (cost/latency) numbers — see [evaluation-plan.md](evaluation-plan.md).
- Replace the hashing embedding with a learned sentence embedding behind the same
  `Featurizer` interface to show the gain compounds (an even stronger φ).
