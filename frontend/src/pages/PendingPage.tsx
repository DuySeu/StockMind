import { Button } from "@/components/ui/button";
import { ArrowLeft, Newspaper } from "lucide-react";
import { Link } from "react-router-dom";

// Render the placeholder for the News route, pointing at what already works
const PendingPage = () => {
  return (
    <div className="mx-auto flex w-full max-w-lg flex-1 flex-col items-center justify-center px-4 py-24 text-center">
      <span className="mb-7 flex size-12 items-center justify-center rounded-lg bg-secondary">
        <Newspaper className="size-6 text-primary" aria-hidden="true" />
      </span>

      <h1 className="mb-3 text-3xl font-bold tracking-tight text-balance">News intelligence is still being built</h1>

      <p className="mb-9 leading-relaxed text-muted-foreground text-pretty">
        Sentiment-scored headlines will land here. Until then the home page carries the latest run, and the assistant
        can pull news for any ticker on request.
      </p>

      <div className="flex flex-col gap-3 sm:flex-row">
        <Button asChild>
          <Link to="/c">Ask the assistant</Link>
        </Button>
        <Button asChild variant="outline">
          <Link to="/#news">
            <ArrowLeft aria-hidden="true" />
            Latest headlines
          </Link>
        </Button>
      </div>
    </div>
  );
};

export default PendingPage;
