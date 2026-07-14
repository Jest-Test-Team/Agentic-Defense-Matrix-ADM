// Command greenteam_agent is the green-team remediation service. It watches the
// battle event stream for attacks that landed and performs remediation: optional
// hosted-LLM triage (severity / revoke / restart targets / SOC summary), then
// revoke + Docker restart of adm.role=agent containers.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/adm/pkg/battle"
	"github.com/adm/pkg/llmops"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/redis/go-redis/v9"
)

type config struct {
	gatewayURL string
	siemURL    string
	redisURL   string
	dryRun     bool
	agentLabel string
	greenLLM   bool
}

func loadConfig() config {
	return config{
		gatewayURL: envOr("ADM_GATEWAY_URL", "http://localhost:8080"),
		siemURL:    envOr("ADM_SIEM_URL", "http://localhost:9091"),
		redisURL:   envOr("ADM_REDIS_URL", "redis://localhost:6379"),
		dryRun:     envOr("ADM_GREEN_DRY_RUN", "false") == "true",
		agentLabel: envOr("ADM_AGENT_LABEL", "adm.role=agent"),
		greenLLM:   llmops.GreenEnabled(),
	}
}

type greenTeam struct {
	cfg     config
	emitter *battle.Emitter
	http    *http.Client
	docker  *client.Client
	rdb     *redis.Client
	llm     *llmops.Client
}

func main() {
	cfg := loadConfig()
	log.Printf("greenteam: gateway=%s siem=%s dry_run=%v label=%q green_llm=%v",
		cfg.gatewayURL, cfg.siemURL, cfg.dryRun, cfg.agentLabel, cfg.greenLLM)

	g := &greenTeam{
		cfg:     cfg,
		emitter: battle.NewEmitter(),
		http:    &http.Client{Timeout: 10 * time.Second},
		llm:     llmops.New(),
	}
	defer g.emitter.Close()

	if !cfg.dryRun {
		if dc, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); err != nil {
			log.Printf("greenteam: docker unavailable (%v); container containment disabled", err)
		} else {
			g.docker = dc
		}
	}

	if opt, err := redis.ParseURL(cfg.redisURL); err == nil {
		g.rdb = redis.NewClient(opt)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go g.pollAlerts(ctx)
	g.watchStream(ctx)
	log.Println("greenteam: shutdown complete")
}

func (g *greenTeam) watchStream(ctx context.Context) {
	if g.rdb == nil {
		log.Println("greenteam: no redis; stream watch disabled")
		<-ctx.Done()
		return
	}
	lastID := "$"
	for {
		if ctx.Err() != nil {
			return
		}
		res, err := g.rdb.XRead(ctx, &redis.XReadArgs{
			Streams: []string{battle.RedisStream, lastID},
			Block:   2 * time.Second,
			Count:   50,
		}).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			time.Sleep(time.Second)
			continue
		}
		for _, stream := range res {
			for _, msg := range stream.Messages {
				lastID = msg.ID
				raw, ok := msg.Values["event"].(string)
				if !ok {
					continue
				}
				var ev battle.Event
				if json.Unmarshal([]byte(raw), &ev) != nil {
					continue
				}
				if ev.Team == battle.TeamRed && ev.Kind == battle.KindAttack &&
					ev.Outcome == battle.OutcomeAllowed {
					g.remediate(ctx, ev)
				}
			}
		}
	}
}

func (g *greenTeam) remediate(ctx context.Context, attack battle.Event) {
	log.Printf("greenteam: remediating session=%s technique=%s target=%s",
		attack.SessionID, attack.Technique, attack.Target)

	triage := llmops.DefaultTriage(attack)
	if g.cfg.greenLLM {
		if tr, err := g.llm.TriageRemediation(ctx, attack); err != nil {
			log.Printf("greenteam: LLM triage failed, using fallback: %v", err)
		} else {
			triage = *tr
		}
	}

	if triage.Revoke {
		g.revokeSession(ctx, attack, triage)
	} else {
		g.emit(ctx, attack, battle.OutcomeRevoked, "skipped",
			"triage: revoke=false — "+triage.Summary, 0, triage)
	}

	g.containAgent(ctx, attack, triage)
}

func (g *greenTeam) revokeSession(ctx context.Context, attack battle.Event, triage llmops.TriageResult) {
	start := time.Now()
	outcome := battle.OutcomeRevoked
	detail := "session revoked on gateway"

	if g.cfg.dryRun {
		detail = "[dry-run] would revoke session"
	} else {
		url := g.cfg.gatewayURL + "/v1/admin/revoke/" + attack.SessionID
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
		if resp, err := g.http.Do(req); err != nil {
			outcome = battle.OutcomeError
			detail = "revoke failed: " + err.Error()
		} else {
			resp.Body.Close()
			if resp.StatusCode >= 400 {
				outcome = battle.OutcomeError
				detail = "revoke returned " + strconv.Itoa(resp.StatusCode)
			}
		}
	}
	if triage.Summary != "" {
		detail = detail + " | " + triage.Summary
	}

	g.emit(ctx, attack, battle.OutcomeRevoked, outcome, detail, time.Since(start).Milliseconds(), triage)
}

func (g *greenTeam) containAgent(ctx context.Context, attack battle.Event, triage llmops.TriageResult) {
	start := time.Now()
	targets := triage.RestartTargets
	if len(targets) == 0 {
		// Nothing to restart — still emit summary for SOC.
		g.emit(ctx, attack, battle.OutcomeRestarted, battle.OutcomeRestarted,
			"triage: no restart targets — "+triage.Summary, time.Since(start).Milliseconds(), triage)
		return
	}

	if g.cfg.dryRun || g.docker == nil {
		detail := "[dry-run] would restart " + strings.Join(targets, ",") + " — " + triage.Summary
		if g.docker == nil && !g.cfg.dryRun {
			detail = "docker unavailable; skipped container containment — " + triage.Summary
		}
		g.emit(ctx, attack, battle.OutcomeRestarted, battle.OutcomeRestarted, detail, time.Since(start).Milliseconds(), triage)
		return
	}

	parts := strings.SplitN(g.cfg.agentLabel, "=", 2)
	f := filters.NewArgs()
	if len(parts) == 2 {
		f.Add("label", parts[0]+"="+parts[1])
	} else {
		f.Add("label", g.cfg.agentLabel)
	}
	containers, err := g.docker.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: f})
	if err != nil {
		g.emit(ctx, attack, battle.OutcomeError, battle.OutcomeError,
			"container list failed: "+err.Error(), time.Since(start).Milliseconds(), triage)
		return
	}

	restarted := 0
	for _, want := range targets {
		for _, c := range containers {
			if !nameMatches(c.Names, want) {
				continue
			}
			timeout := 5
			if err := g.docker.ContainerRestart(ctx, c.ID, container.StopOptions{Timeout: &timeout}); err == nil {
				restarted++
			}
		}
	}
	// Fallback: if triage named a target but none matched, restart all labelled agents.
	if restarted == 0 {
		for _, c := range containers {
			timeout := 5
			if err := g.docker.ContainerRestart(ctx, c.ID, container.StopOptions{Timeout: &timeout}); err == nil {
				restarted++
			}
		}
	}

	outcome := battle.OutcomeRestarted
	detail := "restarted " + strconv.Itoa(restarted) + " agent container(s)"
	if restarted == 0 {
		outcome = battle.OutcomeError
		detail = "no agent containers matched"
	}
	if triage.Summary != "" {
		detail = detail + " | " + triage.Summary
	}
	g.emit(ctx, attack, battle.OutcomeRestarted, outcome, detail, time.Since(start).Milliseconds(), triage)
}

func nameMatches(names []string, target string) bool {
	for _, n := range names {
		if strings.Contains(strings.ToLower(n), strings.ToLower(target)) {
			return true
		}
	}
	return false
}

func (g *greenTeam) pollAlerts(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, g.cfg.siemURL+"/api/v1/alerts", nil)
			resp, err := g.http.Do(req)
			if err != nil {
				continue
			}
			var payload struct {
				Alerts []struct {
					ID       string `json:"id"`
					RuleName string `json:"rule_name"`
					Severity string `json:"severity"`
				} `json:"alerts"`
			}
			json.NewDecoder(resp.Body).Decode(&payload)
			resp.Body.Close()
			for _, a := range payload.Alerts {
				g.emitter.Emit(ctx, &battle.Event{
					Team:      battle.TeamBlue,
					Kind:      battle.KindDefense,
					Technique: a.RuleName,
					SessionID: "siem-" + a.ID,
					Target:    "siem",
					Outcome:   battle.OutcomeDetected,
					Severity:  sevToInt(a.Severity),
					Detail:    "SIEM alert: " + a.RuleName,
				})
			}
		}
	}
}

func (g *greenTeam) emit(ctx context.Context, attack battle.Event, action, outcome, detail string, latency int64, triage llmops.TriageResult) {
	sev := triage.Severity
	if sev < 1 {
		sev = attack.Severity
	}
	labels := map[string]string{
		"action":  action,
		"summary": triage.Summary,
		"triage":  "revoke=" + strconv.FormatBool(triage.Revoke) + ";restart=" + strings.Join(triage.RestartTargets, ","),
	}
	if cid := attack.Labels["chain_id"]; cid != "" {
		labels["chain_id"] = cid
	}
	if step := attack.Labels["chain_step"]; step != "" {
		labels["chain_step"] = step
	}

	g.emitter.Emit(ctx, &battle.Event{
		Team:      battle.TeamGreen,
		Kind:      battle.KindRemediation,
		Technique: attack.Technique,
		Variant:   attack.Variant,
		SessionID: attack.SessionID,
		Target:    attack.Target,
		Outcome:   outcome,
		Severity:  sev,
		LatencyMS: latency,
		Detail:    detail,
		Labels:    labels,
	})
}

func sevToInt(s string) int {
	switch strings.ToLower(s) {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	default:
		return 1
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
