import { Button } from "@/components/ui/button";
import { ArrowLeft, Download, ExternalLink } from "lucide-react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import { PDFGenerator } from "@/lib/pdf-export";

const ResearchResultPage = () => {
  const navigate = useNavigate();
  const { id } = useParams();
  const location = useLocation();

  const response = (location.state as { response: any } | null)?.response;
  const ticker = id?.toUpperCase() ?? "";
  const data: any | undefined = response?.reports?.[ticker];

  if (!response || !data) {
    return (
      <div className="flex flex-col items-center justify-center p-10 gap-4">
        <p className="text-lg text-muted-foreground">No research data available.</p>
        <Button variant="secondary" onClick={() => navigate("/research")}>
          <ArrowLeft /> Go to Research
        </Button>
      </div>
    );
  }

  const tickers = Object.keys(response.reports);

  const handleExportPDF = () => {
    const generator = new PDFGenerator("p");

    // --- Header ---
    generator.addHeader(data.ticker ?? ticker, data.company_name ?? "");

    // --- Market Metrics ---
    generator.addMetricsBoard(data.market_cap, data.pe_ratio);

    // --- Stock Overview ---
    generator.addSectionTitle("Stock Overview");
    if (data.summary) {
      generator.addWrappedText(data.summary);
    }

    // --- Key Insights ---
    if (data.key_insights?.length) {
      generator.addSectionTitle("Key Insights");
      data.key_insights.forEach((insight: string, i: number) => {
        generator.addWrappedText(`${i + 1}. ${insight}`);
      });
    }

    // --- Current Performance ---
    if (data.current_performance) {
      generator.addSectionTitle("Current Performance");
      generator.addWrappedText(data.current_performance);
    }

    // --- Risk Assessment ---
    if (data.risk_assessment) {
      generator.addSectionTitle("Risk Assessment");
      generator.addWrappedText(data.risk_assessment);
    }

    // --- Price Outlook ---
    if (data.price_outlook) {
      generator.addSectionTitle("Price Outlook");
      generator.addWrappedText(data.price_outlook);
    }

    // --- Final Recommendation ---
    if (data.recommendation) {
      generator.addSectionTitle("Final Recommendation");
      generator.addWrappedText(data.recommendation, 11, "bold");
    }

    // --- Sources ---
    generator.addSourceLinks(data.sources);

    // --- Footer ---
    generator.addFooter();

    // --- Save ---
    generator.save(`${ticker}_Research_Report_${new Date().toISOString().slice(0, 10)}.pdf`);
  };

  return (
    <div className="flex flex-col items-center p-3 gap-4 overflow-y-auto">
      <div className="w-full flex justify-between items-center">
        <Button variant="secondary" onClick={() => navigate(-1)}>
          <ArrowLeft /> Back
        </Button>
        <div className="text-2xl font-bold text-primary">
          {id?.toUpperCase()} Research result for {new Date(Date.now()).toLocaleDateString()}
        </div>
        <Button variant="secondary" onClick={() => handleExportPDF()}>
          <Download /> Export PDF
        </Button>
      </div>
      <Select
        defaultValue={ticker}
        onValueChange={(val) => navigate(`/research/${val.toLowerCase()}`, { state: { response } })}
      >
        <SelectTrigger className="w-[220px] text-secondary-foreground">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {tickers.map((t) => (
              <SelectItem key={t} value={t}>
                {t}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
      <div className="flex flex-col items-start w-full max-w-6xl p-3 bg-primary rounded-lg">
        <h1 className="text-2xl font-bold">{data?.ticker}</h1>
        <p className="text-md">{data?.company_name}</p>
        {data.market_cap && data.pe_ratio && (
          <div className="flex gap-3 items-center my-3">
            <div className="p-3 bg-accent rounded-lg w-42">
              <h1 className="text-md font-bold">Market Cap</h1>
              <p className="text-sm">{data?.market_cap}</p>
            </div>
            <div className="p-3 bg-accent rounded-lg w-42">
              <h1 className="text-md font-bold">PE Ratio</h1>
              <p className="text-sm">{data?.pe_ratio}</p>
            </div>
          </div>
        )}
        <div className="p-3 bg-primary/10 border border-border rounded-lg">
          <h1 className="text-md font-bold">Stock Overview</h1>
          <p className="text-sm">{data?.summary}</p>
        </div>
      </div>
      <div className="flex flex-col items-start w-full max-w-6xl p-3 bg-primary rounded-lg">
        <h1 className="text-2xl font-bold">Key Insights</h1>
        <div className="space-y-2">
          {data?.key_insights?.map((item: string, index: number) => (
            <div key={index} className="border border-border rounded-lg p-2">
              <p className="text-sm">{item}</p>
            </div>
          ))}
        </div>
      </div>
      <div className="flex items-start w-full max-w-6xl rounded-lg gap-3">
        <div className="w-calc(100%/3) rounded-lg p-3 bg-primary border border-border">
          <h1 className="text-md font-bold">Current Performance</h1>
          <p className="text-sm p-3 rounded-lg border border-border">{data?.current_performance}</p>
        </div>
        <div className="w-calc(100%/3) rounded-lg p-3 bg-primary border border-border">
          <h1 className="text-md font-bold">Risk Assessment</h1>
          <p className="text-sm p-3 rounded-lg border border-border">{data?.risk_assessment}</p>
        </div>
        <div className="w-calc(100%/3) rounded-lg p-3 bg-primary border border-border">
          <h1 className="text-md font-bold">Price Outlook</h1>
          <p className="text-sm p-3 rounded-lg border border-border">{data?.price_outlook}</p>
        </div>
      </div>
      <div className="flex flex-col items-start w-full max-w-6xl p-3 bg-primary rounded-lg">
        <h1 className="text-2xl font-bold">Final Recommendation</h1>
        <p className="text-sm p-3 rounded-lg border border-border">{data?.recommendation}</p>
      </div>
      <Accordion type="single" collapsible className="w-full max-w-6xl px-3 bg-primary rounded-lg">
        <AccordionItem value="sources">
          <AccordionTrigger>Research Sources ({data?.sources?.length})</AccordionTrigger>
          <AccordionContent className="space-y-2">
            {data?.sources?.map((item: any, index: number) => (
              <a
                key={index}
                href={item.url}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center justify-between text-primary-foreground border border-border rounded-md p-2 hover:bg-accent/10 transition-colors"
              >
                <p className="text-xs">{item.url}</p>
                <ExternalLink className="h-4 w-4 text-accent" />
              </a>
            ))}
          </AccordionContent>
        </AccordionItem>
      </Accordion>
    </div>
  );
};

export default ResearchResultPage;
