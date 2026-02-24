export interface StockReport {
  ticker: string;
  company_name: string;
  summary: string;
  current_performance: string;
  key_insights: string[];
  recommendation: string;
  risk_assessment: string;
  price_outlook: string;
  market_cap: string;
  pe_ratio: string;
  sources: {
    url: string;
    title: string;
  }[];
}

export interface ResearchResponse {
  reports: Record<string, StockReport>;
  generated_at: string;
}
