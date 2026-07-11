package semantic

import "strings"

// Intent-drift detector — the general model from
// docs/research/formalization-intent-drift.md, Eq. 1:
//
//	D_t(W) = (1/W) Σ_{i=t-W+1..t} g( d_M(z_i, C) )   ,   fire when D_t(W) ≥ θ
//
// where z_i = φ(a_i) is the embedded action token, M/C is the authorization
// manifold, and g is a monotone penalty. Holding this pipeline fixed and swapping
// the Featurizer φ is the embedding-φ ablation.

// DriftScorer maps an action token to a per-step drift ∈ [0,1] — how far the
// token has strayed from authorized behavior. It is the per-φ realization of
// d_M(z_t, C): each featurizer measures deviation in its natural way, which is
// itself part of the ablation story.
//
//   - LexicalScorer: drift = attack-keyword mass (the shipped analyzer's
//     semantics). Benign text scores ~0; an *exact*-keyword attack scores high;
//     an obfuscated attack scores ~0 — it is invisible to keywords.
//   - ManifoldScorer: drift = 1 − max cosine to the authorized manifold under a
//     dense embedding φ. Robust to obfuscation, so it stays sensitive where the
//     lexical scorer collapses.
type DriftScorer interface {
	Name() string
	Drift(text string) float64
}

// LexicalScorer is the keyword-mass drift — faithful to the shipped
// analyzer.go: drift = min(1, Σ matched pattern weights).
type LexicalScorer struct{ patterns []Pattern }

// NewLexicalScorer builds the lexical drift from the shared default patterns.
func NewLexicalScorer() *LexicalScorer { return &LexicalScorer{patterns: defaultPatterns()} }

func (s *LexicalScorer) Name() string { return "lexical" }

func (s *LexicalScorer) Drift(text string) float64 {
	low := strings.ToLower(text)
	var score float64
	for _, p := range s.patterns {
		for _, kw := range p.Keywords {
			if strings.Contains(low, strings.ToLower(kw)) {
				score += p.Weight
			}
		}
	}
	if score > 1 {
		score = 1
	}
	return score
}

// ManifoldScorer is the dense-embedding drift: 1 − max cosine to the manifold.
type ManifoldScorer struct {
	f Featurizer
	m *Manifold
}

// NewManifoldScorer wires a dense featurizer to an authorized manifold.
func NewManifoldScorer(f Featurizer, authorized []string) *ManifoldScorer {
	return &ManifoldScorer{f: f, m: NewManifold(f, authorized)}
}

func (s *ManifoldScorer) Name() string              { return s.f.Name() }
func (s *ManifoldScorer) Drift(text string) float64 { return s.m.Drift(s.f.Embed(text)) }

// Manifold is the authorized region C: the set of embedded "legitimate" texts
// (system prompt + authorized tool descriptions + benign in-scope examples).
// Drift of a token is its distance to the nearest authorized anchor.
type Manifold struct {
	anchors []Vec
}

// NewManifold embeds the authorized corpus under φ to form the legitimate
// region. Deduplicated zero vectors are dropped so a φ that produces no signal
// for an anchor (e.g. lexical φ on a benign tool description) doesn't create a
// spurious origin anchor.
func NewManifold(f Featurizer, authorized []string) *Manifold {
	m := &Manifold{}
	for _, s := range authorized {
		v := f.Embed(s)
		if norm2(v) > 0 {
			m.anchors = append(m.anchors, v)
		}
	}
	return m
}

// Drift d_M(z, C) ∈ [0,1]: 1 − max cosine similarity to any authorized anchor.
// A benign in-scope token sits near an anchor (drift → 0); an adversarial token
// is far (drift → 1). If z carries no signal under φ (zero vector), drift is 0 —
// this is exactly why lexical φ *misses* obfuscated payloads: after a base64/
// zero-width mutation the keyword vector is empty, so its drift collapses to 0.
func (m *Manifold) Drift(z Vec) float64 {
	if norm2(z) == 0 || len(m.anchors) == 0 {
		return 0
	}
	best := -1.0
	for _, a := range m.anchors {
		if c := Cosine(z, a); c > best {
			best = c
		}
	}
	d := 1 - best
	if d < 0 {
		d = 0
	}
	return d
}

func norm2(v Vec) float64 {
	var n float64
	for _, x := range v {
		n += x * x
	}
	return n
}

// DriftDetector maintains the sliding window and the windowed statistic D_t(W)
// over a DriftScorer (the per-φ deviation measure).
type DriftDetector struct {
	scorer DriftScorer
	window int       // W
	theta  float64   // θ
	rho    float64   // manifold tube radius; penalty g is a hinge above ρ
	pen    []float64 // ring buffer of recent per-step penalties g(d_M)
	sum    float64   // running sum of the window (for O(1) amortized update)
	head   int
	count  int
	steps  int
}

// NewDriftDetector wires a DriftScorer + (W, θ). rho is the hinge knee for the
// penalty g(u)=max(0, u−ρ); rho=0 uses raw drift as the penalty.
func NewDriftDetector(scorer DriftScorer, window int, theta, rho float64) *DriftDetector {
	if window < 1 {
		window = 1
	}
	return &DriftDetector{
		scorer: scorer, window: window, theta: theta, rho: rho,
		pen: make([]float64, window),
	}
}

// Observation is the per-step detector output.
type Observation struct {
	Step    int     // 1-based step index t
	Drift   float64 // d_M(z_t, C)
	Penalty float64 // g(d_M(z_t, C))
	Stat    float64 // D_t(W) — the windowed statistic
	Fired   bool    // D_t(W) ≥ θ
}

// Observe embeds one action token, updates the window in O(1) amortized time,
// and reports whether the windowed drift statistic crossed θ.
func (d *DriftDetector) Observe(text string) Observation {
	drift := d.scorer.Drift(text)
	pen := drift - d.rho
	if pen < 0 {
		pen = 0
	}

	// O(1) ring-buffer update of the windowed sum.
	if d.count == d.window {
		d.sum -= d.pen[d.head]
	} else {
		d.count++
	}
	d.pen[d.head] = pen
	d.sum += pen
	d.head = (d.head + 1) % d.window
	d.steps++

	stat := d.sum / float64(d.count)
	return Observation{
		Step: d.steps, Drift: drift, Penalty: pen, Stat: stat, Fired: stat >= d.theta,
	}
}

// Reset clears the window (start of a new session/trajectory).
func (d *DriftDetector) Reset() {
	for i := range d.pen {
		d.pen[i] = 0
	}
	d.sum, d.head, d.count, d.steps = 0, 0, 0, 0
}

// RunTrajectory feeds a whole trajectory and returns the first step at which the
// detector fired (0 if it never fired) plus the peak windowed statistic. Used by
// the ablation to measure detection latency (in steps) and FNR.
func (d *DriftDetector) RunTrajectory(traj []string) (fireStep int, peak float64) {
	d.Reset()
	for _, a := range traj {
		o := d.Observe(a)
		if o.Stat > peak {
			peak = o.Stat
		}
		if o.Fired && fireStep == 0 {
			fireStep = o.Step
		}
	}
	return fireStep, peak
}

// AuthorizedCorpus is a small default authorization set (system prompt + tool
// descriptions + benign in-scope example requests) used to seed the manifold in
// tests and the ablation runner. In production this comes from the deployed
// system prompt and the registered tool schemas.
func AuthorizedCorpus() []string {
	return []string{
		"You are a helpful research assistant. Answer the user's questions about the codebase and documentation.",
		"tool: search_docs — search the project documentation for a query string",
		"tool: read_file — read the contents of a file within the workspace",
		"tool: summarize — produce a concise summary of provided text",
		"tool: list_directory — list files under a given path in the workspace",
		"Please summarize the architecture section of the design document.",
		"Search the docs for how the analysis engine ingests battle events.",
		"What does the gateway do when it receives a chat request?",
		"List the files under the deploy directory and read the compose file.",
		"Give me a short summary of the red team corpus generator.",
	}
}
