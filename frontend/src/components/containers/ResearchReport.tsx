import { ExternalLink } from "lucide-react";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";

const ResearchReport = ({ data }: { data: any }) => {
  if (!data) {
    return (
      <div className="flex flex-col items-center justify-center p-10 gap-4">
        <p className="text-lg text-muted-foreground">No research data available.</p>
      </div>
    );
  }

  return (
    <>
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
        <div className="w-full space-y-2">
          {data?.key_insights?.map((item: string, index: number) => (
            <div key={index} className="border border-border rounded-lg p-2">
              <p className="text-sm">{item}</p>
            </div>
          ))}
        </div>
      </div>
      <div className="grid grid-cols-3 w-full max-w-6xl rounded-lg gap-3">
        <div className="rounded-lg p-3 bg-primary border border-border">
          <h1 className="text-md font-bold">Current Performance</h1>
          <p className="text-sm p-3 rounded-lg border border-border">{data?.current_performance}</p>
        </div>
        <div className="rounded-lg p-3 bg-primary border border-border">
          <h1 className="text-md font-bold">Risk Assessment</h1>
          <p className="text-sm p-3 rounded-lg border border-border">{data?.risk_assessment}</p>
        </div>
        <div className="rounded-lg p-3 bg-primary border border-border">
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
    </>
  );
};

export default ResearchReport;
