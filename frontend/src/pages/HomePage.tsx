import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Separator } from "@/components/ui/separator";
import { LogoTile } from "@/components/Logo";
import {
  Search,
  ArrowRight,
  Globe,
  BarChart3,
  Bell,
  LineChart,
  CheckCircle,
  AlertTriangle,
  Zap,
  Check,
} from "lucide-react";
import { Link } from "react-router-dom";
import { useEffect, useState } from "react";
import { getLatestNews } from "@/api/news";
import type { PriceBoard } from "@/types/stock";
import { getPriceBoard } from "@/api/stock";
import { PRICE_STATE, getPriceState } from "@/lib/stock";

/* ───────────────────────── Data ───────────────────────── */

const capabilities = [
  {
    icon: Globe,
    title: "AI Research",
    description: "Instant deep-dives into industry trends and company fundamentals using GPT-4 trained on VN data.",
  },
  {
    icon: BarChart3,
    title: "Financials",
    description: "Auto-summarized quarterly reports. We find the red flags so you don't have to read 200 pages.",
  },
  {
    icon: Bell,
    title: "Smart Watchlist",
    description: "Dynamic tracking with alerts for price breakthroughs, volume spikes, and unusual insider trading.",
  },
  {
    icon: LineChart,
    title: "Macro Insights",
    description: "Understand how SBV policy changes and global macro trends specifically impact your portfolio.",
  },
];

const pricingPlans = [
  {
    name: "Free",
    price: "0",
    features: ["5 AI Queries per day", "Delayed market data", "Basic Watchlist"],
    buttonText: "Get Started",
    popular: false,
  },
  {
    name: "Pro",
    price: "299k",
    features: [
      "Unlimited AI Queries",
      "Real-time Price Alerts",
      "Advanced Financial Analysis",
      "Custom AI Signal Tuning",
    ],
    buttonText: "Upgrade to Pro",
    popular: true,
  },
  {
    name: "Investor",
    price: "999k",
    features: [
      "Priority AI Processing",
      "Personalized Macro Reports",
      "API Access for Algos",
      "1-on-1 AI Training Session",
    ],
    buttonText: "Contact Sales",
    popular: false,
  },
];

/* ───────────────────── Sub-Components ───────────────────── */

function HeroSection() {
  return (
    /* The old background was a hardcoded rgba(160,255,155,…) radial — the
       previous mint green, frozen into the markup and immune to the theme.
       This is the same effect built from tokens. */
    <section className="relative overflow-hidden bg-gradient-to-b from-secondary/70 via-background to-background px-4 pb-24 pt-16 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-3xl text-center">
        <h1 className="mb-5 text-4xl font-bold leading-[1.1] tracking-tight lg:text-5xl">
          Your AI Copilot for <br />
          {/* --accent is a dark surface in dark mode, so the old from-primary
              to-accent gradient faded the second half of the headline into the
              background. chart-3 is a readable violet in both themes. */}
          <span className="bg-gradient-to-r from-primary to-chart-3 bg-clip-text text-transparent">
            Vietnam Stock Investing
          </span>
        </h1>

        <p className="mx-auto mb-8 max-w-2xl text-base text-muted-foreground lg:text-lg">
          Analyze, track, and master the Vietnamese stock market with real-time AI insights, automated research, and
          expert-grade signals.
        </p>

        {/* AI Chat Input */}
        <div className="group relative mx-auto max-w-2xl">
          <div className="glass-raised relative flex items-center gap-1 rounded-xl p-2 focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/30">
            <Search className="ml-2 size-5 shrink-0 text-muted-foreground" aria-hidden="true" />
            <Input
              aria-label="Ask about a stock"
              className="h-auto border-none bg-transparent px-3 py-3 text-base shadow-none placeholder:text-muted-foreground focus-visible:ring-0 dark:bg-transparent"
              placeholder="Ask about FPT, VNM vs MSN, or VIC…"
            />
            <Button asChild className="shrink-0">
              <Link to="/c">Analyze</Link>
            </Button>
          </div>

          <div className="mt-4 flex flex-wrap items-center justify-center gap-2 text-sm text-muted-foreground">
            <span>Try:</span>
            {['"Compare VIC vs VHM financials"', '"FPT technical outlook"', '"Market sentiment today"'].map((q) => (
              <Link
                key={q}
                to="/c"
                className="rounded-md border border-border bg-card px-2.5 py-1 text-xs backdrop-blur-sm transition-colors hover:border-ring hover:text-foreground"
              >
                {q}
              </Link>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}

function TickerCard({ listingInfo, matchPrice, index = 0 }: PriceBoard & { index?: number }) {
  if (!listingInfo || !matchPrice) return null;

  const ticker = listingInfo.symbol;
  const sector = listingInfo.enOrganShortName || "Unknown";

  const priceVal = matchPrice.matchPrice || 0;
  const refPrice = matchPrice.referencePrice || 0;
  const priceDisplay = priceVal.toLocaleString();

  const changeVal = priceVal - refPrice;
  const changePct = refPrice > 0 ? (changeVal / refPrice) * 100 : 0;
  const changePositive = changeVal > 0;
  const changeStr = `${changeVal > 0 ? "+" : ""}${changePct.toFixed(1)}%`;

  const signal = changePositive ? "Buy" : changeVal < 0 ? "Sell" : "Hold";
  const strength = changePositive ? 8.4 : changeVal < 0 ? 3.2 : 5.0;
  const strengthPct = strength * 10;

  // Board state drives the colour, so a card and a board row never disagree.
  const state = getPriceState({
    price: priceVal,
    reference: refPrice,
    ceiling: listingInfo.ceiling,
    floor: listingInfo.floor,
  });
  const style = PRICE_STATE[state];

  return (
    /* `backwards` fill so the card stays hidden through its stagger delay —
       without it the card paints solid, then blinks out to start animating. */
    <Card
      className="gap-0 rounded-xl p-5 animate-in fade-in slide-in-from-bottom-2 duration-300 ease-out"
      style={{ animationDelay: `${index * 60}ms`, animationFillMode: "backwards" }}
    >
      <div className="mb-4 flex items-start justify-between gap-2">
        <div className="min-w-0">
          <h3 className="text-lg font-bold">{ticker}</h3>
          <p className="truncate text-xs text-muted-foreground" title={sector}>
            {sector}
          </p>
        </div>
        <span
          className={`shrink-0 rounded-md px-2 py-0.5 text-xs font-semibold uppercase tracking-wide ${style.bg} ${style.text}`}
        >
          {signal}
        </span>
      </div>

      <div className="mb-4 flex items-baseline gap-2">
        <span className="font-mono text-2xl font-semibold tabular-nums">{priceDisplay}</span>
        {/* Arrow as well as colour — the change is not carried by hue alone. */}
        <span className={`font-mono text-sm font-semibold tabular-nums ${style.text}`}>
          <span aria-hidden="true" className="mr-0.5 text-[10px]">
            {style.sign}
          </span>
          {changeStr}
        </span>
      </div>

      <Progress
        value={strengthPct}
        aria-label={`AI strength ${strength.toFixed(1)} out of 10`}
        className={`h-1 ${state === "up" || state === "ceiling" ? "" : "[&>[data-slot=progress-indicator]]:bg-muted-foreground"}`}
      />

      <div className="mt-2 flex justify-between text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
        <span>AI Strength</span>
        <span className="font-mono tabular-nums">{strength.toFixed(1)}/10</span>
      </div>
    </Card>
  );
}

function WatchlistSection() {
  const [priceBoard, setPriceBoard] = useState<PriceBoard[]>([]);

  useEffect(() => {
    getPriceBoard(4).then((res) => {
      setPriceBoard(res);
    });
  }, []);

  return (
    <section id="watchlist" className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      <div className="mb-6 flex items-center justify-between gap-4">
        <h2 className="text-2xl font-bold tracking-tight">Market Watchlist</h2>
        <Link
          to="/watchlist"
          className="flex shrink-0 items-center gap-1 text-sm font-medium text-primary transition-colors hover:underline"
        >
          View all <ArrowRight className="size-4" aria-hidden="true" />
        </Link>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
        {priceBoard.length > 0
          ? priceBoard.map((d, i) => <TickerCard key={d.listingInfo?.symbol ?? i} index={i} {...d} />)
          : /* Reserved space rather than nothing, so the section does not
               jump when data lands. */
            Array.from({ length: 4 }).map((_, i) => (
              <Card key={i} className="h-[168px] animate-pulse gap-0 rounded-xl bg-muted/60 p-5" />
            ))}
      </div>
    </section>
  );
}

function CapabilitiesSection() {
  return (
    <section id="features" className="border-y border-border bg-card/50 px-4 py-16 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-7xl">
        <div className="mb-12 text-center">
          <h2 className="mb-3 text-3xl font-bold tracking-tight">Unmatched AI Capabilities</h2>
          <p className="text-muted-foreground">The most powerful toolset ever built for HSX and HNX.</p>
        </div>

        <div className="grid grid-cols-1 gap-5 md:grid-cols-2 lg:grid-cols-4">
          {capabilities.map((cap) => (
            <div
              key={cap.title}
              className="glass rounded-xl p-6 transition-colors hover:border-ring"
            >
              <span className="mb-4 inline-flex size-10 items-center justify-center rounded-lg bg-secondary">
                <cap.icon className="size-5 text-primary" aria-hidden="true" />
              </span>
              <h3 className="mb-2 text-base font-semibold">{cap.title}</h3>
              <p className="text-sm leading-relaxed text-muted-foreground">{cap.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function StockAnalysisSection() {
  return (
    <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      <div className="grid grid-cols-1 items-center gap-10 lg:grid-cols-12">
        {/* Stock Card Mockup */}
        <div className="lg:col-span-7">
          <Card className="gap-0 overflow-hidden rounded-xl py-0 shadow-lg">
            <div className="flex flex-wrap items-center justify-between gap-4 border-b border-border p-6">
              <div className="flex items-center gap-3">
                <div className="flex size-11 items-center justify-center rounded-lg bg-secondary text-lg font-bold">
                  F
                </div>
                <div>
                  <h3 className="text-lg font-bold">FPT Corporation</h3>
                  <p className="text-sm text-muted-foreground">HOSE: FPT • Technology</p>
                </div>
              </div>
              <div className="text-right">
                <div className="font-mono text-2xl font-semibold tabular-nums">
                  121,300{" "}
                  <span className="text-base font-semibold text-price-up">
                    <span aria-hidden="true" className="text-[10px]">
                      ▲
                    </span>{" "}
                    +2.3%
                  </span>
                </div>
                <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Market Open</div>
              </div>
            </div>

            <CardContent className="p-6">
              <div className="mb-6 grid grid-cols-3 gap-3">
                <div className="rounded-lg bg-price-up-bg p-3">
                  <p className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-price-up">AI Score</p>
                  <p className="font-mono text-2xl font-bold tabular-nums text-price-up">
                    8.4
                    <span className="text-sm font-normal opacity-70">/10</span>
                  </p>
                </div>
                <div className="rounded-lg bg-muted p-3">
                  <p className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
                    Revenue Growth
                  </p>
                  <p className="font-mono text-2xl font-bold tabular-nums">+21%</p>
                </div>
                <div className="rounded-lg bg-secondary p-3">
                  <p className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
                    Rating
                  </p>
                  <p className="text-lg font-bold text-primary">L-T Buy</p>
                </div>
              </div>

              <div className="space-y-3">
                <div className="flex items-start gap-2.5">
                  <CheckCircle className="mt-0.5 size-4 shrink-0 text-price-up" aria-hidden="true" />
                  <p className="text-sm">Cloud and AI services expansion driving 25% margin growth.</p>
                </div>
                <div className="flex items-start gap-2.5">
                  <CheckCircle className="mt-0.5 size-4 shrink-0 text-price-up" aria-hidden="true" />
                  <p className="text-sm">Strong dividend history with 20% yield growth YoY.</p>
                </div>
                <div className="flex items-start gap-2.5">
                  <AlertTriangle className="mt-0.5 size-4 shrink-0 text-price-ref" aria-hidden="true" />
                  <p className="text-sm">High valuation relative to regional peers (P/E 22.4x).</p>
                </div>
              </div>

              <div className="mt-6">
                <div className="relative flex h-36 w-full items-center justify-center overflow-hidden rounded-lg bg-muted">
                  <LineChart className="size-12 text-muted-foreground/40" aria-hidden="true" />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Chatbot Demo */}
        <div className="lg:col-span-5">
          <div className="space-y-6">
            <div>
              <Badge className="mb-3 rounded-md border-none bg-secondary px-2.5 py-1 text-xs font-semibold uppercase tracking-wide text-secondary-foreground">
                Interactive Demo
              </Badge>
              <h2 className="text-3xl font-bold leading-tight tracking-tight">
                Your Portfolio <br />
                Converses with You
              </h2>
              <p className="mt-3 text-muted-foreground">
                Stop staring at charts. Start asking questions. Our AI understands Vietnamese market nuances and
                specific ticker histories.
              </p>
            </div>

            {/* Mini Chat UI — mirrors the real chat: user filled on the right,
                assistant plain on the left. */}
            <Card className="gap-0 space-y-3 rounded-xl p-5 shadow-md">
              <div className="flex justify-end gap-2.5">
                <div className="max-w-[80%] rounded-xl rounded-tr-sm bg-primary px-3.5 py-2.5 text-sm text-primary-foreground">
                  Why is FPT up today?
                </div>
              </div>

              <div className="flex gap-2.5">
                <div className="flex size-7 shrink-0 items-center justify-center rounded-full bg-secondary">
                  <Zap className="size-3.5 text-primary" aria-hidden="true" />
                </div>
                <div className="text-sm leading-relaxed text-foreground">
                  FPT reported a 20.1% increase in pre-tax profit for the first 5 months. The tech sector is also
                  seeing net foreign buying of over 200bn VND today.
                </div>
              </div>

              <div className="flex items-center gap-2 pt-1 pl-9">
                <span className="flex items-center gap-1" aria-hidden="true">
                  <span className="size-1.5 animate-pulse rounded-full bg-muted-foreground" />
                  <span className="size-1.5 animate-pulse rounded-full bg-muted-foreground [animation-delay:150ms]" />
                  <span className="size-1.5 animate-pulse rounded-full bg-muted-foreground [animation-delay:300ms]" />
                </span>
                <span className="text-xs text-muted-foreground">AI is typing…</span>
              </div>
            </Card>
          </div>
        </div>
      </div>
    </section>
  );
}

function NewsSection() {
  const [localNewsData, setLocalNewsData] = useState<any[]>([]);

  useEffect(() => {
    getLatestNews()
      .then((res) => {
        console.log(res);
        if (Array.isArray(res)) {
          // Process backend News into dynamic frontend elements
          const formatted = res.map((item: any) => {
            const descLower = String(item.description || "").toLowerCase();
            const titleLower = String(item.title || "").toLowerCase();
            const combined = titleLower + " " + descLower;

            // Classification rules unchanged; only the class strings became
            // tokens so the badges follow the dark-mode toggle.
            let sentiment = "Neutral";
            let sentimentColor = "text-price-ref bg-price-ref-bg";
            let impact = "No immediate change";
            let impactColor = "text-muted-foreground";

            if (
              combined.includes("tăng") ||
              combined.includes("tích cực") ||
              combined.includes("lợi nhuận") ||
              combined.includes("chấp thuận")
            ) {
              sentiment = "Positive";
              sentimentColor = "text-price-up bg-price-up-bg";
              impact = "+ Market Sentiment";
              impactColor = "text-price-up";
            } else if (
              combined.includes("giảm") ||
              combined.includes("lỗ") ||
              combined.includes("rủi ro") ||
              combined.includes("cảnh báo")
            ) {
              sentiment = "Negative";
              sentimentColor = "text-price-down bg-price-down-bg";
              impact = "- Market Pressure";
              impactColor = "text-price-down";
            }

            let summary = item.description || "";
            if (summary.length > 150) {
              summary = summary.substring(0, 147) + "...";
            }

            const dt = new Date(item.created_at || new Date());
            const displayTime = isNaN(dt.getTime())
              ? "Today"
              : dt.toLocaleString("vi-VN", { hour: "2-digit", minute: "2-digit", day: "2-digit", month: "2-digit" });

            return {
              sentiment,
              sentimentColor,
              time: displayTime,
              title: item.title,
              summary: "AI Summary: " + summary,
              impact,
              impactColor,
              url: item.url,
            };
          });
          setLocalNewsData(formatted);
        }
      })
      .catch((err) => console.error("Failed to fetch news:", err));
  }, []);

  return (
    <section id="news" className="border-y border-border bg-card/50 px-4 py-16 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-7xl">
        <h2 className="mb-10 text-center text-2xl font-bold tracking-tight">AI News Intelligence</h2>

        <div className="grid grid-cols-1 gap-5 md:grid-cols-3">
          {localNewsData.length > 0 ? (
            localNewsData.map((news: any, i: number) => (
              <a
                href={news.url || "#"}
                target="_blank"
                rel="noopener noreferrer"
                key={i}
                /* animation-duration-*, not duration-*: the latter also retimes
                   the hover lift's transition, doubling it to 300ms. */
                className="group block rounded-xl transition-transform hover:-translate-y-0.5 motion-reduce:hover:translate-y-0 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring animate-in fade-in slide-in-from-bottom-2 animation-duration-300 ease-out"
                style={{ animationDelay: `${i * 60}ms`, animationFillMode: "backwards" }}
              >
                <Card className="h-full gap-0 rounded-xl p-5">
                  <div className="mb-3 flex items-center gap-2">
                    {/* The word carries the sentiment; the colour only reinforces it. */}
                    <span
                      className={`rounded-md px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide ${news.sentimentColor}`}
                    >
                      {news.sentiment}
                    </span>
                    <span className="font-mono text-xs tabular-nums text-muted-foreground">{news.time}</span>
                  </div>

                  <h3 className="mb-3 font-semibold leading-snug group-hover:underline">{news.title}</h3>

                  <div className="mb-3 rounded-lg bg-muted p-3">
                    <p className="line-clamp-3 text-xs leading-relaxed text-muted-foreground">{news.summary}</p>
                  </div>

                  <p className={`mt-auto text-xs font-semibold ${news.impactColor}`}>Impact: {news.impact}</p>
                </Card>
              </a>
            ))
          ) : (
            <div className="col-span-full py-10 text-center text-sm text-muted-foreground">
              Fetching latest AI news…
            </div>
          )}
        </div>
      </div>
    </section>
  );
}

function PricingSection() {
  return (
    <section id="pricing" className="mx-auto max-w-7xl px-4 py-20 sm:px-6 lg:px-8">
      <div className="mb-12 text-center">
        <h2 className="mb-3 text-3xl font-bold tracking-tight">Simple, Transparent Pricing</h2>
        <p className="text-muted-foreground">Choose the plan that fits your investing style.</p>
      </div>

      {/* items-start + a ring instead of scale-105: the popular card used to be
          transformed, which overlapped its neighbours on narrow screens. */}
      <div className="grid grid-cols-1 items-start gap-6 md:grid-cols-3">
        {pricingPlans.map((plan) => (
          <Card
            key={plan.name}
            className={`relative gap-0 rounded-xl p-7 ${
              plan.popular ? "border-primary shadow-lg ring-1 ring-primary md:-mt-2" : ""
            }`}
          >
            {plan.popular && (
              <div className="absolute right-6 top-0 -translate-y-1/2 rounded-md bg-primary px-2.5 py-1 text-xs font-semibold uppercase tracking-wide text-primary-foreground">
                Most Popular
              </div>
            )}

            <h3 className="mb-1.5 text-base font-semibold">{plan.name}</h3>

            <div className="mb-5 flex items-baseline gap-1">
              <span className="font-mono text-3xl font-bold tabular-nums">{plan.price}</span>
              <span className="text-sm text-muted-foreground">VND/mo</span>
            </div>

            <ul className="mb-8 space-y-3 text-sm">
              {plan.features.map((f) => (
                <li key={f} className="flex items-start gap-2">
                  <Check className="mt-0.5 size-4 shrink-0 text-primary" aria-hidden="true" />
                  {f}
                </li>
              ))}
            </ul>

            <Button variant={plan.popular ? "default" : "outline"} className="w-full">
              {plan.buttonText}
            </Button>
          </Card>
        ))}
      </div>
    </section>
  );
}

function Footer() {
  return (
    /* Was bg-foreground + text-muted-foreground — a light-workspace grey on
       near-black, about 2.4:1. The sidebar tokens are the theme's dark surface
       and are contrast-gated as a pair. */
    <footer className="bg-sidebar px-4 py-16 text-sidebar-foreground/70 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-7xl">
        {/* Not a four-column link farm: every link below goes somewhere that
            exists. The old footer held nine href="#" dead ends. */}
        <div className="flex flex-col gap-10 md:flex-row md:items-start md:justify-between">
          <div className="max-w-sm">
            <div className="mb-4 flex items-center gap-2.5">
              <LogoTile className="size-8 shrink-0" />
              <span className="text-lg font-bold tracking-tight text-sidebar-foreground">StockMind</span>
            </div>
            <p className="text-sm leading-relaxed">
              Research tooling for the Vietnamese market, built to read filings and price data in the language they
              were published in.
            </p>
          </div>

          <nav aria-label="Footer" className="grid grid-cols-2 gap-x-12 gap-y-3 text-sm sm:grid-cols-3">
            {[
              { label: "Assistant", to: "/c" },
              { label: "Watchlist", to: "/watchlist" },
              { label: "Research", to: "/research" },
              { label: "Knowledge base", to: "/documents" },
              { label: "Features", to: "/#features" },
              { label: "Pricing", to: "/#pricing" },
            ].map((link) => (
              <Link key={link.label} to={link.to} className="transition-colors hover:text-sidebar-foreground">
                {link.label}
              </Link>
            ))}
          </nav>
        </div>

        <Separator className="my-8 bg-sidebar-border" />

        <div className="flex flex-col gap-3 text-xs md:flex-row md:items-center md:justify-between">
          <p>© 2026 StockMind</p>
          <p className="max-w-md md:text-right">
            Prices from HSX and HNX, delayed. Nothing here is investment advice.
          </p>
        </div>
      </div>
    </footer>
  );
}

/* ────────────────────── Main Page ──────────────────────── */

const HomePage = () => {
  return (
    <div className="w-full flex-1">
      <HeroSection />
      <WatchlistSection />
      <CapabilitiesSection />
      <StockAnalysisSection />
      <NewsSection />
      <PricingSection />
      <Footer />
    </div>
  );
};

export default HomePage;
