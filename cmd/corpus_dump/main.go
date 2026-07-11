// Command corpus_dump renders the deterministic red-team corpus
// (GenerateCorpus with the same size/seed the redteam_agent uses) into a static
// JSON the dashboard ships and pages through, so the "attack matrix" can list
// all enumerated variants (RT-00001 … RT-10000), not just the 30 base classes.
//
// Regenerate: go run ./cmd/corpus_dump > dashboard/public/corpus.json
package main

import (
	"encoding/json"
	"os"
	"strconv"

	"github.com/adm/pkg/redteam"
)

func envIntOr(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// row is the slim, dashboard-facing shape of one variant.
type row struct {
	ID        string `json:"id"`        // enumerated campaign id, RT-00001…
	Technique string `json:"technique"` // base family, RT-001…RT-030
	Name      string `json:"name"`      // human name, e.g. "Reverse Shell"
	Tag       string `json:"tag"`       // OWASP-LLM / MITRE ATLAS tag
	Mutation  string `json:"mutation"`  // how the seed was mutated
	Lang      string `json:"lang"`      // paraphrase language
	Severity  int    `json:"severity"`
	Target    string `json:"target"`
	Preview   string `json:"preview"` // first chars of the payload
}

func main() {
	// Match the redteam_agent defaults (ADM_CORPUS_SIZE / ADM_CORPUS_SEED).
	size := envIntOr("ADM_CORPUS_SIZE", 10000)
	seed := int64(envIntOr("ADM_CORPUS_SEED", 1337))

	corpus := redteam.GenerateCorpus(size, seed)
	rows := make([]row, 0, len(corpus))
	for _, v := range corpus {
		preview := v.Payload
		if len(preview) > 140 {
			preview = preview[:140] + "…"
		}
		rows = append(rows, row{
			ID: v.VariantID, Technique: v.Technique, Name: v.Name, Tag: v.Tag,
			Mutation: v.Mutation, Lang: v.Lang, Severity: v.Severity,
			Target: v.Target, Preview: preview,
		})
	}

	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(map[string]any{"size": len(rows), "seed": seed, "variants": rows}); err != nil {
		panic(err)
	}
}
