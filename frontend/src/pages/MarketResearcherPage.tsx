import { getMarketResearch } from "@/api/stock";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormField, FormItem } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Plus, X } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";

const MarketResearcherPage = () => {
  const navigate = useNavigate();
  const watchList = ["FPT", "HPG", "VCB", "VPB", "TCB", "VNM"];
  const [researchList, setResearchList] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const form = useForm({
    defaultValues: {
      symbols: "",
    },
  });

  const onSubmit = (data: any) => {
    console.log(data);
    form.reset();
    if (researchList.length >= 5) return;
    setResearchList([...researchList, data.symbols.toUpperCase()]);
  };

  const handleResearch = async () => {
    try {
      setIsLoading(true);
      const response = await getMarketResearch(researchList, "mini");
      navigate(`/research/${researchList[0].toLowerCase()}`, {
        state: { response },
      });
    } catch (error) {
      console.error(error);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="flex flex-col items-center p-3 gap-4">
      <span className="text-sm text-accent mt-4 px-3 py-1 rounded-full bg-accent/10">
        Powered by Tavily <strong>/research</strong>
      </span>
      <div className="text-2xl font-bold text-primary">Market Researcher</div>
      <div className="text-sm text-muted-foreground">
        Get comprehensive market insights and analysis for your favorite stocks.
      </div>
      <div className="flex flex-col w-2/3 p-3 gap-4 bg-primary rounded-lg">
        <span className="text-xl text-center font-bold text-primary-foreground">Enter stock Tickers</span>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex gap-2">
            <FormField
              control={form.control}
              name="symbols"
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <Input
                      {...field}
                      className="text-secondary-foreground"
                      placeholder="Enter symbols to research..."
                    />
                  </FormControl>
                </FormItem>
              )}
            />
            <Button type="submit" size="icon" variant="secondary">
              <Plus />
            </Button>
          </form>
        </Form>
        <span className="text-primary-foreground">Watchlist stocks:</span>
        <div className="flex flex-wrap gap-2">
          {watchList.map((stock) => (
            <Button
              key={stock}
              variant="secondary"
              disabled={researchList.includes(stock) || researchList.length >= 5}
              onClick={() => setResearchList([...researchList, stock])}
            >
              {stock}
            </Button>
          ))}
        </div>
        {researchList.length > 0 && (
          <>
            <span className="text-primary-foreground">Selected tickers ({researchList.length}/5):</span>
            <div className="flex flex-wrap gap-2">
              {researchList.map((stock) => (
                <div key={stock} className="flex items-center bg-secondary rounded-md">
                  <span className="text-secondary-foreground p-2 pr-0">{stock}</span>
                  <Button
                    variant="secondary"
                    size="icon-sm"
                    onClick={() => setResearchList(researchList.filter((s) => s !== stock))}
                  >
                    <X />
                  </Button>
                </div>
              ))}
            </div>
          </>
        )}
        <Button disabled={researchList.length === 0 || isLoading} variant="secondary" onClick={handleResearch}>
          Get Daily Digest ({researchList.length} tickers)
        </Button>
        {isLoading && <div className="text-primary-foreground">Loading...</div>}
      </div>
    </div>
  );
};

export default MarketResearcherPage;
