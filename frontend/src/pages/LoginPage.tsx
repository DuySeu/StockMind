import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Sparkles, BarChart3 } from "lucide-react";
import { LogoTile } from "@/components/Logo";
import { Link } from "react-router-dom";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useState } from "react";

/* Sign-in has no backend yet (there is no auth in the API). The form validates
   for real and then says so, instead of a button that silently does nothing. */
const credentialsSchema = z.object({
  email: z.string().min(1, "Enter your email address.").email("That does not look like an email address."),
  password: z.string().min(8, "Passwords are at least 8 characters."),
});

type Credentials = z.infer<typeof credentialsSchema>;

/* ───────────────────── Left: Login Form ───────────────────── */

// Render the credentials form: validated client-side, no sign-in endpoint yet
function LoginForm() {
  const [submitError, setSubmitError] = useState("");

  const form = useForm<Credentials>({
    resolver: zodResolver(credentialsSchema),
    defaultValues: { email: "", password: "" },
  });

  // Validation runs, then the honest failure: the endpoint does not exist.
  const onSubmit = () => {
    setSubmitError("Sign-in is not available yet. StockMind runs without an account for now.");
  };

  return (
    <div className="flex w-full flex-col justify-center bg-card-solid px-6 py-16 sm:px-12 lg:w-1/2 lg:px-24">
      <div className="mx-auto w-full max-w-md">
        <Link
          to="/"
          className="mb-12 inline-flex items-center gap-2.5 rounded-md focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
        >
          <LogoTile className="size-9 shrink-0 rounded-[25%] shadow-xs" />
          <span className="text-xl font-bold tracking-tight">StockMind</span>
        </Link>

        <div className="mb-8">
          <h1 className="mb-2 text-3xl font-bold tracking-tight">Welcome back</h1>
          <p className="text-muted-foreground">Sign in to sync your watchlist and research history.</p>
        </div>

        {/* Providers are not wired either, so they are disabled with a reason
            rather than looking live. */}
        <div className="mb-8 flex flex-col gap-3">
          <Button variant="outline" className="h-11 w-full gap-3 text-sm font-medium" disabled>
            <svg className="size-5" viewBox="0 0 24 24" aria-hidden="true">
              <path
                d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"
                fill="#4285F4"
              />
              <path
                d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                fill="#34A853"
              />
              <path
                d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
                fill="#FBBC05"
              />
              <path
                d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
                fill="#EA4335"
              />
            </svg>
            Continue with Google
          </Button>

          <Button variant="outline" className="h-11 w-full gap-3 text-sm font-medium" disabled>
            <svg className="size-5" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.8-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z" />
            </svg>
            Continue with Apple
          </Button>
        </div>

        <div className="relative mb-8">
          <div className="absolute inset-0 flex items-center">
            <Separator />
          </div>
          <div className="relative flex justify-center">
            <span className="bg-card-solid px-4 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              Or continue with email
            </span>
          </div>
        </div>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5" noValidate>
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="text-sm font-semibold">Email address</FormLabel>
                  <FormControl>
                    <Input {...field} type="email" autoComplete="email" placeholder="name@company.com" className="h-11" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="password"
              render={({ field }) => (
                <FormItem>
                  <div className="flex items-center justify-between">
                    <FormLabel className="text-sm font-semibold">Password</FormLabel>
                    <span className="text-xs text-muted-foreground">Recovery coming soon</span>
                  </div>
                  <FormControl>
                    <Input {...field} type="password" autoComplete="current-password" className="h-11" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Inline, next to the control that failed — never window.alert. */}
            {submitError && (
              <p role="alert" className="rounded-md bg-status-error-bg px-3 py-2 text-sm text-status-error">
                {submitError}
              </p>
            )}

            <Button type="submit" className="h-11 w-full font-semibold">
              Sign in
            </Button>
          </form>
        </Form>

        <p className="mt-8 text-center text-sm text-muted-foreground">
          No account needed to try it —{" "}
          <Link to="/c" className="font-semibold text-primary hover:underline">
            open the assistant
          </Link>
          .
        </p>
      </div>
    </div>
  );
}

/* ───────────────── Right: Visual / Hero Panel ───────────────── */

// Render the decorative right half of the split login screen
function HeroPanel() {
  return (
    /* Was a hardcoded rgba(160,255,155,…) mint radial — the pre-theme accent,
       frozen into the markup. This is the shared --surface-wash instead, so the
       panel follows the dark-mode toggle like every other surface. */
    <aside
      aria-hidden="true"
      className="relative hidden w-1/2 items-center justify-center overflow-hidden bg-background p-24 lg:flex"
      style={{ backgroundImage: "var(--surface-wash)" }}
    >
      <div className="absolute -right-48 -top-48 size-96 rounded-full bg-primary/15 blur-[120px]" />
      <div className="absolute -bottom-48 -left-48 size-96 rounded-full bg-accent/15 blur-[100px]" />

      <div className="relative z-10 max-w-lg text-center">
        <div className="relative mb-14">
          <div className="glass-raised mx-auto flex aspect-square w-full max-w-[380px] items-center justify-center overflow-hidden rounded-3xl">
            <div className="flex size-full flex-col items-center justify-center p-8">
              <BarChart3 className="size-24 text-primary/70" />
              <div className="mt-6 w-full space-y-3">
                <div className="mx-auto h-2 w-3/4 rounded-full bg-primary/40" />
                <div className="mx-auto h-2 w-1/2 rounded-full bg-primary/30" />
                <div className="mx-auto h-2 w-2/3 rounded-full bg-primary/20" />
              </div>
            </div>
          </div>

          {/* Overlapping the card corner rather than sitting beside it — the
              negative offset is what gives the panel any depth. */}
          <Card className="absolute -bottom-6 -right-2 flex-row items-center gap-3 rounded-2xl p-4 shadow-xl xl:-right-6">
            <span className="flex size-10 items-center justify-center rounded-full bg-secondary text-primary">
              <Sparkles className="size-5" />
            </span>
            <div className="text-left">
              <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">VNINDEX</p>
              <p className="font-mono text-sm font-semibold tabular-nums text-price-up">+1,247.62</p>
            </div>
          </Card>
        </div>

        <h2 className="mb-5 text-4xl font-bold leading-[1.1] tracking-tight text-balance">
          Your AI copilot for Vietnam stock investing
        </h2>
        <p className="text-lg leading-relaxed text-muted-foreground text-pretty">
          Read the market in Vietnamese, get answers in either language, and keep the research beside the prices.
        </p>
      </div>
    </aside>
  );
}

/* ────────────────────── Main Page ──────────────────────── */

// Render the split-screen sign-in page
const LoginPage = () => {
  return (
    <div className="flex min-h-dvh w-full bg-background font-sans text-foreground">
      <LoginForm />
      <HeroPanel />
    </div>
  );
};

export default LoginPage;
