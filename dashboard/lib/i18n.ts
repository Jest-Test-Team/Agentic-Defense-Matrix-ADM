// Lightweight i18n for the dashboard: English + Traditional Chinese.
// Strings are written to read naturally, not machine-translated.

export type Lang = "en" | "zh-Hant";

export interface Dict {
  subtitle: string;
  live: string;
  unreachable: string;
  connecting: string;

  aboutTitle: string;
  aboutBody: string[];
  aboutRed: string;
  aboutBlue: string;
  aboutGreen: string;
  aboutHowto: string;

  systemStatus: string;
  svcUp: string;
  svcDown: string;
  svcDisabled: string;
  checking: string;
  cat: Record<string, string>;

  llmTitle: string;
  llmPrimary: string;
  llmFallback: string;
  llmActive: string;
  llmStandby: string;
  llmUnconfigured: string;

  detailsTitle: string;
  close: string;
  clickHint: string;
  hostOnly: string;
  enableHint: string;
  category: string;
  technology: string;
  statusLabel: string;
  landedSessions: string;
  remediatedSessions: string;
  allSessions: string;
  blockedSessions: string;
  detectedSessions: string;
  mttrSessions: string;
  residualSessions: string;
  noneYet: string;

  scoreboard: string;
  attacks: string;
  blockRate: string;
  detectionRate: string;
  landed: string;
  remediations: string;
  mttr: string;
  residualRisk: string;

  liveFeed: string;
  waitingEvents: string;
  noStream: string;
  byTechnique: string;
  blockedLegend: string;
  landedLegend: string;
  loading: string;

  recentSessions: string;
  colTechnique: string;
  colTarget: string;
  colAttack: string;
  colRemediation: string;
  noSessions: string;
  pending: string;
  notNeeded: string;

  attackChains: string;
  chainStrategy: string;
  chainSteps: string;
  chainSummary: string;
  chainStatus: string;
  chainNone: string;
  chainDetail: string;
  chainMutationSource: string;
  chainPayload: string;

  redTeamMatrix: string;
  matrixId: string;
  matrixAttack: string;
  matrixTechnique: string;
  matrixNote: string;
  matrixViewAll: string;
  matrixFullTitle: string;
  matrixIntro: string;
  matrixSearchPlaceholder: string;
  matrixColMutation: string;
  matrixColLang: string;
  matrixColSeverity: string;
  matrixColTarget: string;
  matrixColPreview: string;
  matrixAllFamilies: string;
  matrixShowing: (a: number, b: number, total: number) => string;
  matrixPrev: string;
  matrixNext: string;
  matrixLoading: string;

  searchNav: string;
  searchTitle: string;
  searchIntro: string[];
  searchPlaceholder: string;
  searchButton: string;
  searchExamplesLabel: string;
  searchTotal: (n: number) => string;
  searchNone: string;
  searchDisabled: string;
  backToConsole: string;

  endpoint: string;
  analysisUrl: string;
  gatewayUrl: string;
  save: string;
  orUse: string;

  footNote: string;
  githubLink: string;
  mixedContent: string;
}

export const translations: Record<Lang, Dict> = {
  en: {
    subtitle: "Red attacks · Blue defends · Green remediates",
    live: "live",
    unreachable: "unreachable",
    connecting: "connecting…",

    aboutTitle: "What is this?",
    aboutBody: [
      "Agentic Defense Matrix (ADM) is a defense-in-depth system for AI agents that can plan tasks and call tools. Instead of relying only on prompt filtering, it watches what agents actually do — at the API gateway, the policy engine, and the operating-system level — and contains the blast radius when something goes wrong.",
      "This console shows a live security exercise running on the deployed system:",
    ],
    aboutRed: "Red team continuously attacks the system with thousands of adversarial prompts and tool-call attempts — prompt injection, reverse shells, data exfiltration, container escape, and more. On a landing it can ask the hosted LLM for an adaptive next step and persist a successful attack chain.",
    aboutBlue: "Blue team (the gateway, SIEM, and policy engine) detects the attacks and blocks them at the boundary.",
    aboutGreen: "Green team automatically remediates any attack that slips through — LLM triage decides revoke / which agent to restart, and writes a SOC summary to the dashboard.",
    aboutHowto: "Every event is logged to a database and scored, so you can see in real time how well the defenses hold. The scoreboard below updates every few seconds; the feed on the left streams each attack and defense as it happens.",

    systemStatus: "System status",
    svcUp: "up",
    svcDown: "down",
    svcDisabled: "disabled",
    checking: "checking…",
    llmTitle: "LLM backend (Groq → X.AI failover)",
    llmPrimary: "Primary · Groq",
    llmFallback: "Fallback · X.AI",
    llmActive: "IN USE",
    llmStandby: "stand-by",
    llmUnconfigured: "not configured",
    detailsTitle: "Details",
    close: "Close",
    clickHint: "click for details",
    hostOnly: "host-only",
    enableHint: "enable on A1",
    category: "Category",
    technology: "Technology",
    statusLabel: "Status",
    landedSessions: "Attacks that landed (recent)",
    remediatedSessions: "Remediated sessions (recent)",
    allSessions: "Recent attacks (all outcomes)",
    blockedSessions: "Blocked attacks (recent)",
    detectedSessions: "Detected attacks (recent)",
    mttrSessions: "Remediations with time-to-fix (recent)",
    residualSessions: "Residual risk — landed & not yet remediated",
    noneYet: "None in the recent window.",
    cat: { Edge: "Edge", Detection: "Detection", Agents: "Agents", Runtime: "Runtime", Data: "Data", Ops: "Ops" },

    scoreboard: "Battle scoreboard",
    attacks: "Attacks",
    blockRate: "Block rate",
    detectionRate: "Detection rate",
    landed: "Landed",
    remediations: "Remediations",
    mttr: "Mean time to remediate",
    residualRisk: "Residual risk",

    liveFeed: "Live battle feed",
    waitingEvents: "Waiting for events…",
    noStream: "No connection to the event stream.",
    byTechnique: "By technique — blocked ▏landed",
    blockedLegend: "blocked (blue)",
    landedLegend: "landed (red)",
    loading: "Loading…",

    recentSessions: "Recent sessions (attack → remediation)",
    colTechnique: "technique",
    colTarget: "target",
    colAttack: "attack",
    colRemediation: "remediation / MTTR",
    noSessions: "No sessions yet.",
    pending: "pending",
    notNeeded: "not needed",

    attackChains: "Successful attack chains",
    chainStrategy: "strategy",
    chainSteps: "steps",
    chainSummary: "remediation summary",
    chainStatus: "status",
    chainNone: "No successful attack chains yet.",
    chainDetail: "Attack chain detail",
    chainMutationSource: "source",
    chainPayload: "payload",

    redTeamMatrix: "Red team attack matrix",
    matrixId: "ID",
    matrixAttack: "Attack",
    matrixTechnique: "Technique",
    matrixNote: "These 30 rows are the base attack classes. At runtime the red team deterministically expands them into 10,000 enumerated campaign variants (RT-00001 … RT-10000). On a landing, adaptive LLM mutation can append follow-up steps to an attack chain.",
    matrixViewAll: "Browse all 10,000 variants",
    matrixFullTitle: "Attack matrix — all 10,000 variants",
    matrixIntro: "Every enumerated variant the red team fires, generated deterministically (seed 1337). Each derives from one of the 30 base techniques via a mutation + language paraphrase. Search by id, technique, name, tag, mutation, or payload text.",
    matrixSearchPlaceholder: "search id / technique / name / tag / mutation / payload…",
    matrixColMutation: "Mutation",
    matrixColLang: "Lang",
    matrixColSeverity: "Sev",
    matrixColTarget: "Target",
    matrixColPreview: "Payload preview",
    matrixAllFamilies: "All base techniques",
    matrixShowing: (a, b, total) => `Showing ${a.toLocaleString()}–${b.toLocaleString()} of ${total.toLocaleString()}`,
    matrixPrev: "‹ Prev",
    matrixNext: "Next ›",
    matrixLoading: "Loading corpus…",

    searchNav: "🔎 Search events",
    searchTitle: "Event search (Elasticsearch)",
    searchIntro: [
      "Every attack, defense, and remediation is also indexed into an Elasticsearch cluster (a free Bonsai 'Hobby' cluster). The scoreboard on the home page is powered by Postgres and answers fixed questions — counts, rates, MTTR. Elasticsearch answers the open-ended ones: full-text search across every event's payload and metadata, in milliseconds, over the whole history.",
      "Type a query below (Lucene syntax). It searches the adm-battle-events index and returns the newest matches — the same thing you can run by hand in Bonsai's Console tab.",
    ],
    searchPlaceholder: 'e.g. reverse shell   ·   team:red AND outcome:allowed   ·   technique:RT-028',
    searchButton: "Search",
    searchExamplesLabel: "Try:",
    searchTotal: (n: number) => `${n.toLocaleString()} matching events`,
    searchNone: "No events matched. Try a broader query, or * for everything.",
    searchDisabled: "Search is unavailable — the analysis engine has no ELASTIC_URL configured.",
    backToConsole: "← Back to battle console",

    endpoint: "Endpoint",
    analysisUrl: "analysis API base URL",
    gatewayUrl: "gateway base URL",
    save: "Save & reload",
    orUse: "or use ?api=…&gw=… in the URL",

    footNote:
      "Polls the analysis API every few seconds and streams live events over Server-Sent Events. Durable log in Neon Postgres; search and aggregation in Elasticsearch.",
    githubLink: "GitHub repository",
    mixedContent:
      "Live data is blocked by the browser (mixed content): this page is HTTPS but the API endpoint is HTTP. Point the dashboard at an HTTPS endpoint with ?api=https://your-host, or use the Endpoint box below.",
  },
  "zh-Hant": {
    subtitle: "紅隊攻擊 · 藍隊防守 · 綠隊修復",
    live: "連線中",
    unreachable: "無法連線",
    connecting: "連線中…",

    aboutTitle: "這是什麼？",
    aboutBody: [
      "Agentic Defense Matrix（ADM，代理式防禦矩陣）是一套為「會自己規劃任務、會呼叫工具的 AI 代理」所設計的縱深防禦系統。它不只靠過濾提示詞，而是實際觀察代理的行為——在 API 閘道、政策引擎，以及作業系統層——並在出問題時，把影響範圍控制到最小。",
      "這個看板呈現一場正在運行的即時攻防演練：",
    ],
    aboutRed: "紅隊 持續用上千種對抗性提示與工具呼叫攻擊系統——提示注入、反向 shell、資料外洩、容器逃逸等等。攻擊落地後可用託管 LLM 做 adaptive 下一步，並把成功攻擊鏈寫入資料庫。",
    aboutBlue: "藍隊（閘道、SIEM、政策引擎）負責偵測攻擊，並在邊界就把它們攔下來。",
    aboutGreen: "綠隊 會自動修復任何漏網的攻擊——LLM triage 決定是否撤銷連線、重啟哪個代理，並產生給 SOC 的修復摘要。",
    aboutHowto: "每一筆事件都會寫入資料庫並計分，讓你即時看到防禦守得有多穩。下方的計分板每幾秒更新一次；左側的即時動態會把每一次攻擊與防禦即時串流出來。",

    systemStatus: "系統狀態",
    svcUp: "運行中",
    svcDown: "離線",
    svcDisabled: "未啟用",
    checking: "檢查中…",
    llmTitle: "語言模型後端（Groq → X.AI 自動切換）",
    llmPrimary: "主要 · Groq",
    llmFallback: "備援 · X.AI",
    llmActive: "使用中",
    llmStandby: "待命中",
    llmUnconfigured: "未設定",
    detailsTitle: "詳細資訊",
    close: "關閉",
    clickHint: "點擊查看詳情",
    hostOnly: "僅限主機端",
    enableHint: "可於 A1 啟用",
    category: "分類",
    technology: "技術",
    statusLabel: "狀態",
    landedSessions: "成功穿透的攻擊（近期）",
    remediatedSessions: "已修復的工作階段（近期）",
    allSessions: "近期攻擊（所有結果）",
    blockedSessions: "已攔截的攻擊（近期）",
    detectedSessions: "已偵測的攻擊（近期）",
    mttrSessions: "含修復時間的修復紀錄（近期）",
    residualSessions: "殘餘風險 — 已穿透且尚未修復",
    noneYet: "近期區間內沒有資料。",
    cat: { Edge: "邊界", Detection: "偵測", Agents: "代理", Runtime: "執行環境", Data: "資料", Ops: "維運" },

    scoreboard: "攻防計分板",
    attacks: "攻擊次數",
    blockRate: "攔截率",
    detectionRate: "偵測率",
    landed: "成功穿透",
    remediations: "修復次數",
    mttr: "平均修復時間",
    residualRisk: "殘餘風險",

    liveFeed: "即時攻防動態",
    waitingEvents: "等待事件中…",
    noStream: "無法連上事件串流。",
    byTechnique: "各攻擊手法 — 攔截 ▏穿透",
    blockedLegend: "已攔截（藍）",
    landedLegend: "已穿透（紅）",
    loading: "載入中…",

    recentSessions: "近期連線（攻擊 → 修復）",
    colTechnique: "手法",
    colTarget: "目標",
    colAttack: "攻擊結果",
    colRemediation: "修復 / 修復時間",
    noSessions: "尚無連線紀錄。",
    pending: "等待修復",
    notNeeded: "無需修復",

    attackChains: "成功攻擊鏈",
    chainStrategy: "策略",
    chainSteps: "步驟",
    chainSummary: "修復摘要",
    chainStatus: "狀態",
    chainNone: "尚無成功攻擊鏈。",
    chainDetail: "攻擊鏈詳情",
    chainMutationSource: "來源",
    chainPayload: "攻擊內容",

    redTeamMatrix: "紅隊攻擊矩陣",
    matrixId: "ID",
    matrixAttack: "攻擊",
    matrixTechnique: "技術",
    matrixNote: "這 30 列是「基礎攻擊類別」。實際運行時，紅隊會以決定性方式展開成 10,000 個變體；攻擊落地後，可用 LLM adaptive mutation 在同一攻擊鏈上追加後續步驟。",
    matrixViewAll: "瀏覽全部 10,000 個變體",
    matrixFullTitle: "攻擊矩陣 — 全部 10,000 個變體",
    matrixIntro: "紅隊實際發動的每一個列舉變體，以決定性方式產生（種子 1337）。每個變體都源自 30 個基礎手法之一，經過一次變異與語言改寫。可用 id、技術、名稱、標籤、變異方式或攻擊內容搜尋。",
    matrixSearchPlaceholder: "搜尋 id／技術／名稱／標籤／變異／攻擊內容…",
    matrixColMutation: "變異方式",
    matrixColLang: "語言",
    matrixColSeverity: "嚴重度",
    matrixColTarget: "目標",
    matrixColPreview: "攻擊內容預覽",
    matrixAllFamilies: "所有基礎手法",
    matrixShowing: (a, b, total) => `顯示第 ${a.toLocaleString()}–${b.toLocaleString()} 筆，共 ${total.toLocaleString()} 筆`,
    matrixPrev: "‹ 上一頁",
    matrixNext: "下一頁 ›",
    matrixLoading: "載入攻擊語料中…",

    searchNav: "🔎 搜尋事件",
    searchTitle: "事件搜尋（Elasticsearch）",
    searchIntro: [
      "每一筆攻擊、防禦與修復事件，也會同步索引進一個 Elasticsearch 叢集（免費的 Bonsai「Hobby」方案）。首頁的計分板由 Postgres 驅動，負責回答固定的問題——次數、比率、修復時間；Elasticsearch 則負責回答開放式問題：在毫秒之內，對整段歷史裡每一筆事件的內容與中繼資料進行全文檢索。",
      "在下方輸入查詢（Lucene 語法），它會搜尋 adm-battle-events 索引並回傳最新的符合結果——這與你在 Bonsai 的 Console 分頁手動執行的查詢是同一件事。",
    ],
    searchPlaceholder: '例如 reverse shell　·　team:red AND outcome:allowed　·　technique:RT-028',
    searchButton: "搜尋",
    searchExamplesLabel: "試試：",
    searchTotal: (n: number) => `${n.toLocaleString()} 筆符合的事件`,
    searchNone: "沒有符合的事件。試試更寬鬆的查詢，或用 * 查看全部。",
    searchDisabled: "搜尋暫時無法使用——分析引擎未設定 ELASTIC_URL。",
    backToConsole: "← 返回攻防主控台",

    endpoint: "連線端點",
    analysisUrl: "分析 API 網址",
    gatewayUrl: "閘道網址",
    save: "儲存並重新載入",
    orUse: "或在網址加上 ?api=…&gw=…",

    footNote:
      "每幾秒輪詢一次分析 API，並透過 Server-Sent Events 串流即時事件。永久紀錄存放在 Neon Postgres；搜尋與彙整由 Elasticsearch 負責。",
    githubLink: "GitHub 原始碼",
    mixedContent:
      "瀏覽器擋下了即時資料（混合內容）：本頁是 HTTPS，但 API 端點是 HTTP。請用 ?api=https://你的主機 指向 HTTPS 端點，或使用下方的連線端點欄位。",
  },
};

export function getLang(): Lang {
  if (typeof window === "undefined") return "en";
  const saved = window.localStorage.getItem("adm_lang");
  if (saved === "en" || saved === "zh-Hant") return saved;
  return navigator.language.startsWith("zh") ? "zh-Hant" : "en";
}

export function setLang(l: Lang) {
  if (typeof window !== "undefined") window.localStorage.setItem("adm_lang", l);
}
