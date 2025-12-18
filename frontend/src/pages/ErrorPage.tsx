import { Link, useRouteError, isRouteErrorResponse } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { AlertCircle, Home, RefreshCw } from "lucide-react";

const ErrorPage = () => {
  const error = useRouteError();

  let errorTitle = "Something went wrong";
  let errorMessage = "An unexpected error occurred.";
  let is404 = false;

  if (isRouteErrorResponse(error)) {
    if (error.status === 404) {
      errorTitle = "Page Not Found";
      errorMessage = "Sorry, we couldn't find the page you're looking for.";
      is404 = true;
    } else {
      errorTitle = `Error ${error.status}`;
      errorMessage = error.statusText;
    }
  } else if (error instanceof Error) {
    errorMessage = error.message;
  } else if (!error) {
    // Fallback if rendered directly or as a catch-all for 404
    errorTitle = "Page Not Found";
    errorMessage = "Sorry, we couldn't find the page you're looking for.";
    is404 = true;
  }

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-background px-4 py-12 text-center">
      <div className="flex flex-col items-center space-y-6 max-w-md">
        <div className="rounded-full bg-muted p-4">
          <AlertCircle className="h-12 w-12 text-destructive" />
        </div>

        <div className="space-y-2">
          <h1 className="text-3xl font-bold tracking-tighter sm:text-4xl text-foreground">{errorTitle}</h1>
          <p className="text-muted-foreground text-lg">{errorMessage}</p>
        </div>

        <div className="flex flex-col sm:flex-row gap-4 min-w-[200px] justify-center">
          <Button asChild size="lg" className="w-full sm:w-auto">
            <Link to="/">
              <Home className="mr-2 h-4 w-4" />
              Go to Home
            </Link>
          </Button>
          {!is404 && (
            <Button variant="outline" size="lg" onClick={() => window.location.reload()} className="w-full sm:w-auto">
              <RefreshCw className="mr-2 h-4 w-4" />
              Try Again
            </Button>
          )}
        </div>
      </div>
    </div>
  );
};

export default ErrorPage;
