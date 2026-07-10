# ADM API HTTPS proxy (Cloudflare Worker)

The GitHub Pages dashboard is HTTPS; the ADM API on the OCI box is HTTP, so the
browser blocks the calls (mixed content). This Worker fronts the API over HTTPS
and adds CORS, so the dashboard can reach it. Free tier is plenty.

## Deploy (one-time)

```bash
cd worker
npm install
npx wrangler login          # opens a browser; authorize your Cloudflare account
npm run deploy
```

`wrangler deploy` prints the URL, e.g. `https://adm-api-proxy.<your-subdomain>.workers.dev`.

## Point the dashboard at it

Open the dashboard with the Worker URL (analysis and gateway are the same host —
the Worker routes `/v1/*` to the gateway and everything else to the analysis API):

```
https://jest-test-team.github.io/Agentic-Defense-Matrix-ADM/?api=https://adm-api-proxy.<sub>.workers.dev&gw=https://adm-api-proxy.<sub>.workers.dev
```

…or paste that URL into the dashboard's **Endpoint** box and click *Save & reload*.
The choice persists in `localStorage`, so afterwards the bare dashboard URL works.

## Why `.nip.io`

Cloudflare Workers **cannot `fetch()` a raw IP** (it returns error 1003). `nip.io`
is wildcard DNS that maps `<ip>.nip.io` → `<ip>` with zero setup, giving the
Worker a hostname to fetch. That's why the vars use
`http://161.33.209.244.nip.io:8090`, not the bare IP.

## When the OCI IP changes

Each `replace_instance` deploy gives the box a new public IP. Update the embedded
IP and redeploy:

```bash
npx wrangler deploy \
  --var ADM_ANALYSIS_URL:http://<new-ip>.nip.io:8090 \
  --var ADM_GATEWAY_URL:http://<new-ip>.nip.io:8080
```

(or edit `vars` in `wrangler.jsonc`). A reserved OCI public IP or a real domain in
front of the box avoids this.
