package semantic

import (
	"hash/fnv"
	"math"
	"strings"
)

// Featurizer is the pluggable semantic map φ: A → ℝ^d from the intent-drift
// formalization (docs/research/formalization-intent-drift.md). Swapping the
// implementation while holding the windowed detector fixed is the ablation that
// separates the *windowing* gain from the *embedding* gain.
//
// Two implementations ship:
//   - LexicalFeaturizer: sparse keyword-indicator φ — the degenerate instance
//     that mirrors the shipped analyzer.go. Blind to obfuscation.
//   - HashEmbeddingFeaturizer: dense char-n-gram feature-hashing φ — a
//     deterministic, offline distributional embedding that survives the corpus's
//     mutation classes (base64, zero-width, case-flip, paraphrase).
type Featurizer interface {
	Name() string
	Dim() int
	// Embed returns the L2-normalized feature vector for a text (zero vector if
	// the text carries no signal in this φ's space).
	Embed(text string) Vec
}

// Vec is a dense feature vector.
type Vec []float64

// Cosine similarity of two equal-length vectors; 0 if either has zero norm.
func Cosine(a, b Vec) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func l2normalize(v Vec) Vec {
	var n float64
	for _, x := range v {
		n += x * x
	}
	if n == 0 {
		return v
	}
	inv := 1 / math.Sqrt(n)
	for i := range v {
		v[i] *= inv
	}
	return v
}

// ---- Lexical φ (the degenerate, keyword-indicator instance) ------------------

// LexicalFeaturizer maps a text onto a fixed axis per known attack keyword,
// weighted by the pattern weight. It fires only on exact (case-insensitive)
// substring presence — so any obfuscating mutation drives the vector to zero.
type LexicalFeaturizer struct {
	keywords []string
	weights  []float64
}

// NewLexicalFeaturizer builds the lexical φ from the same patterns the shipped
// analyzer uses, so the ablation's "keyword-φ" arm is faithful to production.
func NewLexicalFeaturizer() *LexicalFeaturizer {
	patterns := defaultPatterns()
	lf := &LexicalFeaturizer{}
	for _, p := range patterns {
		for _, kw := range p.Keywords {
			lf.keywords = append(lf.keywords, strings.ToLower(kw))
			lf.weights = append(lf.weights, p.Weight)
		}
	}
	return lf
}

func (lf *LexicalFeaturizer) Name() string { return "lexical" }
func (lf *LexicalFeaturizer) Dim() int     { return len(lf.keywords) }

func (lf *LexicalFeaturizer) Embed(text string) Vec {
	low := strings.ToLower(text)
	v := make(Vec, len(lf.keywords))
	for i, kw := range lf.keywords {
		if strings.Contains(low, kw) {
			v[i] = lf.weights[i]
		}
	}
	return l2normalize(v)
}

// ---- Embedding φ (dense char-n-gram feature hashing) -------------------------

// HashEmbeddingFeaturizer is a deterministic, dependency-free distributional
// embedding: it hashes overlapping character n-grams into a fixed-dimensional
// space with signed hashing (the "hashing trick"). Because it represents a text
// by its sub-word shape rather than exact tokens, it stays close to the
// authorized manifold for benign paraphrases and far for adversarial payloads
// even after base64/hex/zero-width/case mutations — which is precisely where the
// lexical φ collapses to zero.
type HashEmbeddingFeaturizer struct {
	dim int
	n   int // n-gram length
}

// NewHashEmbeddingFeaturizer returns a char-n-gram hashing embedding. dim=256,
// n=3 are robust offline defaults; both are reproducible (no learned weights).
func NewHashEmbeddingFeaturizer(dim, n int) *HashEmbeddingFeaturizer {
	if dim <= 0 {
		dim = 256
	}
	if n <= 0 {
		n = 3
	}
	return &HashEmbeddingFeaturizer{dim: dim, n: n}
}

func (h *HashEmbeddingFeaturizer) Name() string { return "embedding" }
func (h *HashEmbeddingFeaturizer) Dim() int     { return h.dim }

func (h *HashEmbeddingFeaturizer) Embed(text string) Vec {
	// Normalize casing and collapse whitespace so surface noise doesn't dominate
	// the sub-word distribution; the n-grams still capture obfuscation structure.
	r := []rune(strings.ToLower(strings.Join(strings.Fields(text), " ")))
	v := make(Vec, h.dim)
	if len(r) < h.n {
		// Fall back to whole-string features for very short inputs.
		if len(r) > 0 {
			bucket, sign := h.hash(string(r))
			v[bucket] += sign
		}
		return l2normalize(v)
	}
	for i := 0; i+h.n <= len(r); i++ {
		gram := string(r[i : i+h.n])
		bucket, sign := h.hash(gram)
		v[bucket] += sign
	}
	return l2normalize(v)
}

// hash maps an n-gram to (bucket, ±1) using two independent FNV hashes; the sign
// hash reduces the collision bias of the hashing trick.
func (h *HashEmbeddingFeaturizer) hash(gram string) (int, float64) {
	f := fnv.New32a()
	_, _ = f.Write([]byte(gram))
	bucket := int(f.Sum32()) % h.dim
	if bucket < 0 {
		bucket += h.dim
	}
	s := fnv.New32()
	_, _ = s.Write([]byte("sign:" + gram))
	if s.Sum32()&1 == 0 {
		return bucket, 1
	}
	return bucket, -1
}
