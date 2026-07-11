// Command ablation runs the embedding-φ ablation from
// docs/research/formalization-intent-drift.md. It holds the windowed drift
// detector D_t(W) fixed and swaps the drift scorer (lexical keyword-mass vs
// dense char-n-gram manifold embedding), reporting detection rate per corpus
// mutation class at a matched benign false-positive rate.
//
// Headline: the lexical scorer (the shipped detector) collapses on obfuscating
// mutations (base64/hex/zero-width/homoglyph/multilang) because the keyword mass
// goes to zero, while the embedding scorer stays sensitive at the same benign
// FPR — the ablation win. A second experiment shows the *windowing* gain: on
// multi-step camouflage where each step is individually sub-threshold, D_t(W)
// with W>1 catches what per-step (W=1) misses.
//
//	go run ./cmd/ablation                 # human table
//	go run ./cmd/ablation -json > r.json  # machine-readable
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/adm/pkg/redteam"
	"github.com/adm/pkg/semantic"
)

func main() {
	var (
		jsonOut  bool
		sample   int
		targetFP float64
		windowW  int
	)
	flag.BoolVar(&jsonOut, "json", false, "emit JSON instead of a table")
	flag.IntVar(&sample, "sample", 4000, "attack variants to evaluate")
	flag.Float64Var(&targetFP, "fpr", 0.02, "target benign false-positive rate to calibrate θ")
	flag.IntVar(&windowW, "window", 6, "sliding window W for the camouflage experiment")
	flag.Parse()

	authorized := semantic.AuthorizedCorpus()
	benign := benignEvalCorpus()
	corpus := redteam.GenerateCorpus(sample, 1337)

	scorers := []semantic.DriftScorer{
		semantic.NewLexicalScorer(),
		semantic.NewManifoldScorer(semantic.NewHashEmbeddingFeaturizer(256, 3), authorized),
	}

	report := Report{TargetFPR: targetFP, Window: windowW, Attacks: len(corpus), Benign: len(benign)}
	for _, s := range scorers {
		report.Arms = append(report.Arms, evaluate(s, benign, corpus, targetFP))
	}
	report.Camouflage = camouflageExperiment(scorers[1], benign, corpus, targetFP, windowW)

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
		return
	}
	printTable(report)
}

// evaluate calibrates θ on the benign set to the target FPR, then measures
// single-step detection over the attack corpus grouped by mutation.
func evaluate(s semantic.DriftScorer, benign []string, corpus []redteam.AttackVariant, targetFP float64) Arm {
	benignDrift := make([]float64, len(benign))
	for i, b := range benign {
		benignDrift[i] = s.Drift(b)
	}
	theta := calibrate(benignDrift, targetFP)

	fp := 0
	for _, d := range benignDrift {
		if d >= theta {
			fp++
		}
	}
	arm := Arm{Scorer: s.Name(), Theta: round(theta), BenignFPR: rate(fp, len(benign))}

	byMut := map[string]*mut{}
	detected := 0
	for _, v := range corpus {
		mm := byMut[v.Mutation]
		if mm == nil {
			mm = &mut{}
			byMut[v.Mutation] = mm
		}
		mm.total++
		if s.Drift(v.Payload) >= theta {
			mm.hit++
			detected++
		}
	}
	arm.Detect = rate(detected, len(corpus))
	for name, mm := range byMut {
		arm.ByMutation = append(arm.ByMutation, MutRate{Mutation: name, Total: mm.total, Detect: rate(mm.hit, mm.total)})
	}
	sort.Slice(arm.ByMutation, func(i, j int) bool { return arm.ByMutation[i].Mutation < arm.ByMutation[j].Mutation })
	return arm
}

// camouflageExperiment demonstrates the *windowing* gain, correctly. A windowed
// mean statistic can only beat per-step detection because averaging reduces the
// variance of the benign statistic (Eq. 2: FPR decays exp in W), letting the
// windowed detector run a *lower* θ_W at the same benign FPR. So we calibrate
// θ_1 (per-step) and θ_W (windowed) SEPARATELY to the same FPR, then measure
// sustained-moderate camouflage — each step carries partial drift below θ_1 but
// the sustained windowed mean crosses the lower θ_W.
func camouflageExperiment(s semantic.DriftScorer, benign []string, corpus []redteam.AttackVariant, targetFP float64, W int) Camouflage {
	// θ_1 from the per-step benign drift distribution.
	benignStep := make([]float64, len(benign))
	for i, b := range benign {
		benignStep[i] = s.Drift(b)
	}
	theta1 := calibrate(benignStep, targetFP)

	// θ_W from the benign *windowed-statistic* distribution: run the windowed
	// detector over benign trajectories and collect every D_t(W).
	var benignStat []float64
	collector := semantic.NewDriftDetector(s, W, math.Inf(1), 0) // never fires
	for _, traj := range benignTrajectories(benign, W) {
		collector.Reset()
		for _, step := range traj {
			benignStat = append(benignStat, collector.Observe(step).Stat)
		}
	}
	thetaW := calibrate(benignStat, targetFP)

	// Higher-severity variants as camouflage seeds.
	var seeds []redteam.AttackVariant
	for _, v := range corpus {
		if v.Severity >= 4 {
			seeds = append(seeds, v)
		}
		if len(seeds) >= 800 {
			break
		}
	}

	d1 := semantic.NewDriftDetector(s, 1, theta1, 0)
	dW := semantic.NewDriftDetector(s, W, thetaW, 0)
	var per, win int
	for _, v := range seeds {
		traj := sustainedCamouflage(v.Payload, benign, W, v.VariantID)
		if fire, _ := d1.RunTrajectory(traj); fire > 0 {
			per++
		}
		if fire, _ := dW.RunTrajectory(traj); fire > 0 {
			win++
		}
	}
	return Camouflage{
		Window: W, Theta1: round(theta1), ThetaW: round(thetaW), Trajectories: len(seeds),
		PerStepDetect: rate(per, len(seeds)), WindowedDetect: rate(win, len(seeds)),
	}
}

// sustainedCamouflage spreads an attack payload across all W steps, each step a
// benign carrier + one payload fragment, so every step carries similar *moderate*
// drift (below θ_1) while the whole window sustains it (crossing the lower θ_W).
func sustainedCamouflage(payload string, benign []string, W int, seed string) []string {
	words := strings.Fields(payload)
	if len(words) == 0 {
		words = []string{payload}
	}
	traj := make([]string, 0, W)
	h := hashSeed(seed)
	per := (len(words) + W - 1) / W
	for i := 0; i < W; i++ {
		lo := i * per
		hi := lo + per
		if lo > len(words) {
			lo = len(words)
		}
		if hi > len(words) {
			hi = len(words)
		}
		frag := strings.Join(words[lo:hi], " ")
		carrier := benign[(h+i)%len(benign)]
		traj = append(traj, strings.TrimSpace(carrier+" "+frag))
	}
	return traj
}

// benignTrajectories chunks the benign corpus into length-W trajectories.
func benignTrajectories(benign []string, W int) [][]string {
	var out [][]string
	for i := 0; i+W <= len(benign); i += W {
		out = append(out, benign[i:i+W])
	}
	return out
}

// calibrate returns θ that admits ~targetFP of the benign drifts as positives:
// θ is set just above the (allowed+1)-th largest benign drift, so an all-zero
// benign set (e.g. the lexical scorer) yields θ just above 0 rather than 0.
func calibrate(benign []float64, targetFP float64) float64 {
	if len(benign) == 0 {
		return 1e-9
	}
	sorted := append([]float64(nil), benign...)
	sort.Float64s(sorted)
	allowed := int(math.Round(targetFP * float64(len(sorted))))
	if allowed >= len(sorted) {
		allowed = len(sorted) - 1
	}
	knee := sorted[len(sorted)-1-allowed] // largest benign we must NOT admit
	return knee + 1e-9
}

// ---- reporting types ---------------------------------------------------------

type Report struct {
	TargetFPR  float64    `json:"target_fpr"`
	Window     int        `json:"window"`
	Attacks    int        `json:"attacks"`
	Benign     int        `json:"benign"`
	Arms       []Arm      `json:"arms"`
	Camouflage Camouflage `json:"camouflage"`
}

type Arm struct {
	Scorer     string    `json:"scorer"`
	Theta      float64   `json:"theta"`
	BenignFPR  float64   `json:"benign_fpr"`
	Detect     float64   `json:"detect"`
	ByMutation []MutRate `json:"by_mutation"`
}

type MutRate struct {
	Mutation string  `json:"mutation"`
	Total    int     `json:"total"`
	Detect   float64 `json:"detect"`
}

type Camouflage struct {
	Window         int     `json:"window"`
	Theta1         float64 `json:"theta_w1"`
	ThetaW         float64 `json:"theta_wW"`
	Trajectories   int     `json:"trajectories"`
	PerStepDetect  float64 `json:"detect_per_step_w1"`
	WindowedDetect float64 `json:"detect_windowed"`
}

type mut struct{ total, hit int }

func printTable(r Report) {
	fmt.Printf("embedding-φ ablation  (attacks=%d, benign=%d, target FPR=%.0f%%)\n", r.Attacks, r.Benign, r.TargetFPR*100)
	fmt.Print("windowed detector D_t(W) held fixed; only the drift scorer φ swapped.\n\n")
	fmt.Printf("%-11s %7s %10s %10s   detection by mutation class →\n", "scorer φ", "θ", "benignFPR", "detect")
	for _, a := range r.Arms {
		fmt.Printf("%-11s %7.3f %9.1f%% %9.1f%%\n", a.Scorer, a.Theta, a.BenignFPR*100, a.Detect*100)
	}
	fmt.Printf("\n%-14s", "mutation")
	for _, a := range r.Arms {
		fmt.Printf(" %12s", a.Scorer)
	}
	fmt.Println("  Δ(embed−lex)")
	muts := r.Arms[0].ByMutation
	lex := indexByMut(r.Arms[0])
	emb := indexByMut(r.Arms[len(r.Arms)-1])
	for _, m := range muts {
		l, e := lex[m.Mutation], emb[m.Mutation]
		fmt.Printf("%-14s %11.1f%% %11.1f%%  %+8.1f pts\n", m.Mutation, l*100, e*100, (e-l)*100)
	}

	c := r.Camouflage
	fmt.Printf("\nwindowing gain (embedding φ, sustained camouflage, %d trajectories, matched FPR):\n", c.Trajectories)
	fmt.Printf("  per-step  W=1  θ₁=%.3f : %.1f%%\n", c.Theta1, c.PerStepDetect*100)
	fmt.Printf("  windowed  W=%d  θ_W=%.3f : %.1f%%   (%+.1f pts — variance reduction lets θ_W < θ₁)\n",
		c.Window, c.ThetaW, c.WindowedDetect*100, (c.WindowedDetect-c.PerStepDetect)*100)

	fmt.Println("\nRead: lexical φ craters on obfuscating mutations (base64/hex/zero-width/homoglyph/")
	fmt.Println("multilang) where the keyword mass is zero; embedding φ stays sensitive at equal")
	fmt.Println("benign FPR. Windowing recovers multi-step camouflage that per-step scoring dilutes.")
}

func indexByMut(a Arm) map[string]float64 {
	m := map[string]float64{}
	for _, x := range a.ByMutation {
		m[x.Mutation] = x.Detect
	}
	return m
}

// ---- helpers -----------------------------------------------------------------

func rate(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}
func round(x float64) float64 { return math.Round(x*1000) / 1000 }
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func hashSeed(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

func benignEvalCorpus() []string {
	verbs := []string{"summarize", "explain", "list", "show", "describe", "find", "read", "compare", "outline", "review"}
	objs := []string{
		"the deployment docs", "the analysis engine's ingest path", "the gateway health check",
		"the red team corpus generator", "the SIEM correlation rules", "the terraform variables",
		"the dashboard components", "the docker compose services", "the OPA policy bundle",
		"the watchdog configuration", "the battle event schema", "the Neon database setup",
		"the CI workflow steps", "the Rust analysis handlers", "the semantic analyzer",
	}
	var out []string
	for _, v := range verbs {
		for _, o := range objs {
			out = append(out, v+" "+o)
		}
	}
	return out
}
