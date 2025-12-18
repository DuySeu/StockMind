import { Button } from "@/components/ui/button";
import { ArrowLeft, Construction } from "lucide-react";
import { Link } from "react-router-dom";

const ExplorePage = () => {
  return (
    <div className="flex h-full min-h-[60vh] flex-col items-center justify-center p-8 text-center animate-in fade-in zoom-in duration-500">
      <div className="rounded-full bg-muted p-6 mb-6">
        <Construction className="h-12 w-12 text-primary" />
      </div>

      <h1 className="text-3xl font-bold tracking-tight mb-2">Work in Progress</h1>

      <p className="text-muted-foreground max-w-[500px] mb-8 text-lg">
        This page is currently pending development. Check back soon for updates!
      </p>

      <Button asChild>
        <Link to="/">
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Home
        </Link>
      </Button>
    </div>
  );
};

export default ExplorePage;
