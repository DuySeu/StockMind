import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Separator } from "@/components/ui/separator";
import {
  TrendingUp,
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
    <section className="relative pt-20 pb-32 px-6 overflow-hidden bg-[radial-gradient(circle_at_50%_50%,rgba(160,255,155,0.1)_0%,var(--background)_100%)]">
      <div className="max-w-4xl mx-auto text-center">
        <h1 className="text-5xl lg:text-[64px] font-black leading-[1.1] tracking-tight mb-6">
          Your AI Copilot for <br />
          <span className="text-transparent bg-clip-text bg-gradient-to-r from-emerald-600 to-green-400">
            Vietnam Stock Investing
          </span>
        </h1>

        <p className="text-lg lg:text-xl text-muted-foreground mb-10 max-w-2xl mx-auto">
          Analyze, track, and master the Vietnamese stock market with real-time AI insights, automated research, and
          expert-grade signals.
        </p>

        {/* AI Chat Input */}
        <div className="relative group max-w-2xl mx-auto">
          {/* Glow */}
          <div className="absolute -inset-1 bg-gradient-to-r from-primary to-emerald-400 rounded-2xl blur opacity-25 group-focus-within:opacity-50 transition duration-1000" />

          <div className="relative flex items-center bg-card border border-border rounded-xl p-2 shadow-xl">
            <Search className="ml-4 size-5 text-muted-foreground" />
            <Input
              className="border-none shadow-none focus-visible:ring-0 px-4 py-4 h-auto text-base placeholder:text-muted-foreground bg-transparent"
              placeholder="Ask about FPT, VNM vs MSN, or VIC..."
            />
            <Button className="rounded-lg px-8 py-3 h-auto font-bold hover:scale-[1.02] transition-transform">
              Analyze
            </Button>
          </div>

          <div className="mt-4 flex flex-wrap justify-center gap-3 text-sm text-muted-foreground">
            <span>Try:</span>
            {['"Compare VIC vs VHM financials"', '"FPT technical outlook"', '"Market sentiment today"'].map((q) => (
              <button key={q} className="hover:text-primary transition-colors">
                {q}
              </button>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}

function TickerCard({ listingInfo, matchPrice }: PriceBoard) {
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

  const signalVariant = changePositive ? "buy" : changeVal < 0 ? "sell" : "neutral";
  const signal = changePositive ? "Buy" : changeVal < 0 ? "Sell" : "Hold";
  const strength = changePositive ? 8.4 : changeVal < 0 ? 3.2 : 5.0;
  const strengthPct = strength * 10;

  const signalBadgeClass =
    signalVariant === "buy"
      ? "bg-primary/20 text-emerald-700 dark:text-emerald-400"
      : signalVariant === "sell"
        ? "bg-rose-500/20 text-rose-700 dark:text-rose-400"
        : "bg-secondary text-muted-foreground";

  const changeClass = changePositive
    ? "text-emerald-500"
    : changePositive === false
      ? "text-rose-500"
      : "text-muted-foreground";

  return (
    <Card className="p-6 rounded-2xl gap-0">
      <div className="flex justify-between items-start mb-4">
        <div>
          <h4 className="text-xl font-black">{ticker}</h4>
          <p className="text-xs text-muted-foreground truncate max-w-[120px]" title={sector}>
            {sector}
          </p>
        </div>
        <span className={`${signalBadgeClass} px-2 py-1 rounded text-xs font-bold uppercase tracking-wider`}>
          {signal}
        </span>
      </div>

      <div className="mb-4">
        <span className="text-2xl font-bold">{priceDisplay}</span>
        <span className={`${changeClass} text-sm font-bold ml-2`}>{changeStr}</span>
      </div>

      <Progress
        value={strengthPct}
        className={`h-1 ${signalVariant === "buy" ? "" : "[&>[data-slot=progress-indicator]]:bg-muted-foreground"}`}
      />

      <div className="flex justify-between mt-2 text-[10px] text-muted-foreground font-bold uppercase">
        <span>AI Strength</span>
        <span>{strength.toFixed(1)}/10</span>
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
    <section id="watchlist" className="py-20 px-6 max-w-7xl mx-auto">
      <div className="flex items-center justify-between mb-10">
        <h3 className="text-2xl font-bold">Market Watchlist</h3>
        <Link
          to="/watchlist"
          className="text-primary font-semibold flex items-center gap-1 hover:opacity-80 transition-opacity"
        >
          View All <ArrowRight className="size-4" />
        </Link>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {priceBoard.map((d, i) => (
          <TickerCard key={d.listingInfo?.symbol ?? i} {...d} />
        ))}
      </div>
    </section>
  );
}

function CapabilitiesSection() {
  return (
    <section id="features" className="py-20 px-6 bg-card/50">
      <div className="max-w-7xl mx-auto">
        <div className="text-center mb-16">
          <h2 className="text-4xl font-black mb-4">Unmatched AI Capabilities</h2>
          <p className="text-muted-foreground">The most powerful toolset ever built for HSX and HNX.</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
          {capabilities.map((cap) => (
            <div
              key={cap.title}
              className="p-8 rounded-3xl bg-background border border-primary/10 hover:border-primary/30 transition-colors"
            >
              <cap.icon className="size-9 text-emerald-500 mb-6" />
              <h3 className="text-xl font-bold mb-3">{cap.title}</h3>
              <p className="text-sm text-muted-foreground">{cap.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function StockAnalysisSection() {
  return (
    <section className="py-20 px-6 max-w-7xl mx-auto">
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-12 items-center">
        {/* Stock Card Mockup */}
        <div className="lg:col-span-7">
          <Card className="rounded-[2.5rem] shadow-2xl overflow-hidden gap-0 py-0">
            {/* Header */}
            <div className="p-8 border-b border-border flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className="size-12 rounded-full bg-secondary flex items-center justify-center font-black text-xl">
                  F
                </div>
                <div>
                  <h4 className="text-2xl font-black">FPT Corporation</h4>
                  <p className="text-sm text-muted-foreground">HOSE: FPT • Technology</p>
                </div>
              </div>
              <div className="text-right">
                <div className="text-3xl font-black">
                  121,300 <span className="text-emerald-500 text-lg font-bold">+2.3%</span>
                </div>
                <div className="text-xs text-muted-foreground font-bold uppercase tracking-widest">Market Open</div>
              </div>
            </div>

            {/* Body */}
            <CardContent className="p-8">
              {/* Stats */}
              <div className="grid grid-cols-3 gap-6 mb-8">
                <div className="bg-emerald-50 dark:bg-emerald-900/20 p-4 rounded-2xl">
                  <p className="text-[10px] text-emerald-600 dark:text-emerald-400 font-black uppercase mb-1">
                    AI Score
                  </p>
                  <p className="text-3xl font-black text-emerald-700 dark:text-emerald-300">
                    8.4
                    <span className="text-sm opacity-60">/10</span>
                  </p>
                </div>
                <div className="bg-secondary/50 p-4 rounded-2xl">
                  <p className="text-[10px] text-muted-foreground font-black uppercase mb-1">Revenue Growth</p>
                  <p className="text-3xl font-black">+21%</p>
                </div>
                <div className="bg-primary/20 p-4 rounded-2xl">
                  <p className="text-[10px] text-emerald-700 font-black uppercase mb-1">Rating</p>
                  <p className="text-xl font-black text-emerald-900 dark:text-primary">L-T Buy</p>
                </div>
              </div>

              {/* Insights */}
              <div className="space-y-4">
                <div className="flex items-center gap-3">
                  <CheckCircle className="size-5 text-emerald-500 shrink-0" />
                  <p className="text-sm font-medium">Cloud and AI services expansion driving 25% margin growth.</p>
                </div>
                <div className="flex items-center gap-3">
                  <CheckCircle className="size-5 text-emerald-500 shrink-0" />
                  <p className="text-sm font-medium">Strong dividend history with 20% yield growth YoY.</p>
                </div>
                <div className="flex items-center gap-3">
                  <AlertTriangle className="size-5 text-amber-500 shrink-0" />
                  <p className="text-sm font-medium">High valuation relative to regional peers (P/E 22.4x).</p>
                </div>
              </div>

              {/* Chart placeholder */}
              <div className="mt-8">
                <div className="w-full h-40 bg-secondary rounded-2xl relative overflow-hidden flex items-center justify-center">
                  <LineChart className="size-16 text-muted-foreground/30" />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Chatbot Demo */}
        <div className="lg:col-span-5">
          <div className="space-y-8">
            <div>
              <Badge className="bg-primary/20 text-emerald-700 dark:text-emerald-400 border-none px-4 py-1.5 rounded-full text-xs font-black uppercase tracking-widest mb-4">
                Interactive Demo
              </Badge>
              <h2 className="text-4xl font-black leading-tight">
                Your Portfolio <br />
                Converses with You
              </h2>
              <p className="text-muted-foreground mt-4">
                Stop staring at charts. Start asking questions. Our AI understands Vietnamese market nuances and
                specific ticker histories.
              </p>
            </div>

            {/* Mini Chat UI */}
            <Card className="rounded-3xl p-6 shadow-xl gap-0 space-y-4">
              {/* User message */}
              <div className="flex gap-3">
                <div className="size-8 rounded-full bg-secondary shrink-0" />
                <div className="bg-secondary p-3 rounded-2xl rounded-tl-none text-sm">Why is FPT up today?</div>
              </div>

              {/* AI response */}
              <div className="flex gap-3 justify-end">
                <div className="bg-primary/20 text-emerald-900 dark:text-primary p-3 rounded-2xl rounded-tr-none text-sm max-w-[80%]">
                  FPT reported a 20.1% increase in pre-tax profit for the first 5 months. The tech sector is also seeing
                  net foreign buying of over 200bn VND today.
                </div>
                <div className="size-8 rounded-full bg-primary shrink-0 flex items-center justify-center">
                  <Zap className="size-4 text-primary-foreground" />
                </div>
              </div>

              {/* Typing indicator */}
              <div className="pt-2">
                <div className="flex items-center bg-secondary rounded-xl px-4 py-2 text-xs text-muted-foreground italic">
                  AI is typing...
                </div>
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

            let sentiment = "Neutral";
            let sentimentColor = "text-amber-500 bg-amber-50 dark:bg-amber-900/20";
            let impact = "No immediate change";
            let impactColor = "text-muted-foreground";

            if (
              combined.includes("tăng") ||
              combined.includes("tích cực") ||
              combined.includes("lợi nhuận") ||
              combined.includes("chấp thuận")
            ) {
              sentiment = "Positive";
              sentimentColor = "text-emerald-500 bg-emerald-50 dark:bg-emerald-900/20";
              impact = "+ Market Sentiment";
              impactColor = "text-primary";
            } else if (
              combined.includes("giảm") ||
              combined.includes("lỗ") ||
              combined.includes("rủi ro") ||
              combined.includes("cảnh báo")
            ) {
              sentiment = "Negative";
              sentimentColor = "text-rose-500 bg-rose-50 dark:bg-rose-900/20";
              impact = "- Market Pressure";
              impactColor = "text-destructive";
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
    <section id="news" className="py-20 px-6 bg-background">
      <div className="max-w-7xl mx-auto">
        <h2 className="text-3xl font-black mb-12 text-center">AI News Intelligence</h2>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {localNewsData.length > 0 ? (
            localNewsData.map((news: any, i: number) => (
              <a
                href={news.url || "#"}
                target="_blank"
                rel="noopener noreferrer"
                key={i}
                className="block transition-transform hover:-translate-y-1"
              >
                <Card className="p-6 rounded-3xl gap-0 h-full">
                  <div className="flex items-center gap-2 mb-4">
                    <span className={`text-[10px] font-black uppercase px-2 py-0.5 rounded ${news.sentimentColor}`}>
                      {news.sentiment}
                    </span>
                    <span className="text-xs text-muted-foreground">{news.time}</span>
                  </div>

                  <h3 className="font-bold mb-3 leading-snug">{news.title}</h3>

                  <div className="bg-secondary p-3 rounded-xl mb-4">
                    <p className="text-xs text-muted-foreground italic line-clamp-3">{news.summary}</p>
                  </div>

                  <p className={`text-[10px] font-bold ${news.impactColor}`}>Impact: {news.impact}</p>
                </Card>
              </a>
            ))
          ) : (
            <div className="col-span-3 text-center text-muted-foreground py-10">Fetching latest AI news...</div>
          )}
        </div>
      </div>
    </section>
  );
}

function PricingSection() {
  return (
    <section id="pricing" className="py-24 px-6 max-w-7xl mx-auto">
      <div className="text-center mb-16">
        <h2 className="text-4xl font-black mb-4">Simple, Transparent Pricing</h2>
        <p className="text-muted-foreground">Choose the plan that fits your investing style.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
        {pricingPlans.map((plan) => (
          <Card
            key={plan.name}
            className={`p-10 rounded-[2.5rem] relative gap-0 ${
              plan.popular ? "bg-foreground text-background scale-105 border-4 border-primary/30 shadow-2xl" : ""
            }`}
          >
            {plan.popular && (
              <div className="absolute top-0 right-10 -translate-y-1/2 bg-primary text-primary-foreground px-4 py-1 rounded-full text-xs font-black uppercase tracking-widest">
                Most Popular
              </div>
            )}

            <h3 className="text-xl font-bold mb-2">{plan.name}</h3>

            <div className="flex items-baseline gap-1 mb-6">
              <span className="text-4xl font-black">{plan.price}</span>
              <span className={plan.popular ? "text-background/50" : "text-muted-foreground"}>VND/mo</span>
            </div>

            <ul className="space-y-4 mb-10 text-sm">
              {plan.features.map((f) => (
                <li key={f} className="flex items-center gap-2">
                  <Check className="size-5 text-primary shrink-0" />
                  {f}
                </li>
              ))}
            </ul>

            <Button
              variant={plan.popular ? "default" : "outline"}
              className={`w-full py-4 h-auto rounded-2xl font-bold ${
                plan.popular ? "shadow-lg shadow-primary/20" : ""
              }`}
            >
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
    <footer className="bg-foreground text-muted-foreground py-20 px-6 border-t border-border">
      <div className="max-w-7xl mx-auto">
        <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-5 gap-12">
          {/* Brand */}
          <div className="col-span-2 lg:col-span-2">
            <div className="flex items-center gap-2 mb-6">
              <div className="size-8 bg-primary rounded-lg flex items-center justify-center">
                <TrendingUp className="size-4 text-primary-foreground" strokeWidth={3} />
              </div>
              <h2 className="text-xl font-black tracking-tight text-background">StockMind</h2>
            </div>
            <p className="max-w-xs text-sm leading-relaxed">
              Empowering the next generation of Vietnamese investors with state-of-the-art AI technology.
            </p>
          </div>

          {/* Links */}
          <div>
            <h4 className="text-background font-bold mb-6">Product</h4>
            <ul className="space-y-4 text-sm">
              {["Features", "Pricing", "Changelog"].map((l) => (
                <li key={l}>
                  <a href="#" className="hover:text-primary transition-colors">
                    {l}
                  </a>
                </li>
              ))}
            </ul>
          </div>

          <div>
            <h4 className="text-background font-bold mb-6">Company</h4>
            <ul className="space-y-4 text-sm">
              {["About Us", "Careers", "Privacy"].map((l) => (
                <li key={l}>
                  <a href="#" className="hover:text-primary transition-colors">
                    {l}
                  </a>
                </li>
              ))}
            </ul>
          </div>

          <div>
            <h4 className="text-background font-bold mb-6">Social</h4>
            <ul className="space-y-4 text-sm">
              {["Facebook", "LinkedIn", "TikTok"].map((l) => (
                <li key={l}>
                  <a href="#" className="hover:text-primary transition-colors">
                    {l}
                  </a>
                </li>
              ))}
            </ul>
          </div>
        </div>

        <Separator className="my-8 bg-background/10" />

        <div className="flex flex-col md:flex-row justify-between items-center gap-4 text-xs">
          <p>© 2024 StockMind AI. All rights reserved.</p>
          <p>Data provided by HSX/HNX. AI insights are for informational purposes only.</p>
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
