import { Button } from "@/components/ui/button";
import { ArrowLeft, Download } from "lucide-react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { PDFGenerator } from "@/lib/pdf-export";
import ResearchReport from "@/components/containers/ResearchReport";

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
      <ResearchReport data={data} />
    </div>
  );
};

export default ResearchResultPage;
