import { Link, useRouteError, isRouteErrorResponse } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { ArrowLeft, Home, RefreshCw } from "lucide-react";
import { LogoTile } from "@/components/Logo";

// Render the route error boundary, branded and with three ways back
const ErrorPage = () => {
  const error = useRouteError();

  let errorCode = "";
  let errorTitle = "Something went wrong";
  let errorMessage = "The page failed to load. Trying again usually fixes it.";
  let is404 = false;

  if (isRouteErrorResponse(error)) {
    if (error.status === 404) {
      errorCode = "404";
      errorTitle = "We couldn't find that page";
      errorMessage = "The link may be out of date, or the ticker in the URL no longer exists.";
      is404 = true;
    } else {
      errorCode = String(error.status);
      errorTitle = "The server rejected that request";
      errorMessage = error.statusText || "No further detail was returned.";
    }
  } else if (error instanceof Error) {
    errorMessage = error.message;
  } else if (!error) {
    // Rendered directly, or reached as the catch-all for an unmatched path.
    errorCode = "404";
    errorTitle = "We couldn't find that page";
    errorMessage = "The link may be out of date, or the ticker in the URL no longer exists.";
    is404 = true;
  }

  return (
    <div className="flex min-h-dvh flex-col items-center justify-center bg-background px-4 py-16">
      {/* Branded rather than an anonymous alert circle — a dead end is still a
          page of the product, and it needs a way back into it. */}
      <Link
        to="/"
        className="mb-12 inline-flex items-center gap-2.5 rounded-md focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
      >
        <LogoTile className="size-9 shrink-0 rounded-[25%] shadow-xs" />
        <span className="text-xl font-bold tracking-tight">StockMind</span>
      </Link>

      <div className="w-full max-w-lg text-center">
        {errorCode && (
          <p className="mb-4 font-mono text-6xl font-bold tabular-nums text-muted-foreground/40">{errorCode}</p>
        )}

        <h1 className="mb-3 text-3xl font-bold tracking-tight text-balance sm:text-4xl">{errorTitle}</h1>
        <p className="mx-auto max-w-md leading-relaxed text-muted-foreground text-pretty">{errorMessage}</p>

        <div className="mt-9 flex flex-col justify-center gap-3 sm:flex-row">
          <Button asChild size="lg">
            <Link to="/">
              <Home aria-hidden="true" />
              Back to home
            </Link>
          </Button>
          {is404 ? (
            <Button asChild variant="outline" size="lg">
              <Link to="/c">Ask the assistant instead</Link>
            </Button>
          ) : (
            <Button variant="outline" size="lg" onClick={() => window.location.reload()}>
              <RefreshCw aria-hidden="true" />
              Try again
            </Button>
          )}
        </div>

        {/* A third, quieter route out — not every dead end wants a filled button. */}
        <button
          type="button"
          onClick={() => window.history.back()}
          className="mx-auto mt-8 flex items-center gap-1.5 rounded-md text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="size-4" aria-hidden="true" />
          Go back to the previous page
        </button>
      </div>
    </div>
  );
};

export default ErrorPage;
