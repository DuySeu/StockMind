import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { TrendingUp, Sparkles, BarChart } from "lucide-react";
import { Link } from "react-router-dom";

/* ───────────────────── Left: Login Form ───────────────────── */

function LoginForm() {
  return (
    <div className="flex flex-col w-full lg:w-1/2 p-8 md:p-16 lg:p-24 justify-center bg-card">
      <div className="max-w-md w-full mx-auto">
        {/* Logo */}
        <div className="flex items-center gap-2 mb-12">
          <div className="text-primary">
            <TrendingUp className="size-8" strokeWidth={2.5} />
          </div>
          <Link to="/" className="text-2xl font-bold tracking-tight">
            StockMind
          </Link>
        </div>

        {/* Header */}
        <div className="mb-8">
          <h1 className="text-3xl font-bold mb-2">Welcome back</h1>
          <p className="text-muted-foreground">Enter your credentials to access your account</p>
        </div>

        {/* Social Logins */}
        <div className="flex flex-col gap-3 mb-8">
          <Button variant="outline" className="w-full h-12 rounded-xl font-medium gap-3 text-sm">
            <svg className="size-5" viewBox="0 0 24 24">
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
            <span>Continue with Google</span>
          </Button>

          <Button variant="outline" className="w-full h-12 rounded-xl font-medium gap-3 text-sm">
            <svg className="size-5" fill="currentColor" viewBox="0 0 24 24">
              <path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.8-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z" />
            </svg>
            <span>Continue with Apple</span>
          </Button>
        </div>

        {/* Divider */}
        <div className="relative mb-8">
          <div className="absolute inset-0 flex items-center">
            <Separator />
          </div>
          <div className="relative flex justify-center text-sm">
            <span className="px-4 bg-card text-muted-foreground uppercase tracking-wider text-xs font-semibold">
              Or continue with email
            </span>
          </div>
        </div>

        {/* Form */}
        <form className="space-y-5" onSubmit={(e) => e.preventDefault()}>
          <div>
            <Label htmlFor="email" className="mb-1.5 text-sm font-semibold">
              Email Address
            </Label>
            <Input id="email" type="email" placeholder="name@company.com" className="h-12 px-4 rounded-xl" />
          </div>

          <div>
            <div className="flex justify-between mb-1.5">
              <Label htmlFor="password" className="text-sm font-semibold">
                Password
              </Label>
              <a href="#" className="text-xs font-semibold text-primary hover:underline">
                Forgot password?
              </a>
            </div>
            <Input id="password" type="password" placeholder="••••••••" className="h-12 px-4 rounded-xl" />
          </div>

          <Button type="submit" className="w-full h-12 rounded-xl font-bold mt-4 shadow-sm">
            Sign In
          </Button>
        </form>

        {/* Footer */}
        <p className="mt-8 text-center text-sm text-muted-foreground">
          Don't have an account?
          <a href="#" className="text-primary font-bold hover:underline ml-1">
            Sign up
          </a>
        </p>
      </div>
    </div>
  );
}

/* ───────────────── Right: Visual / Hero Panel ───────────────── */

function HeroPanel() {
  return (
    <div className="hidden lg:flex w-1/2 bg-[radial-gradient(circle_at_center,rgba(160,255,155,0.15)_0%,var(--background)_100%)] relative overflow-hidden items-center justify-center p-24">
      {/* Decorative blurs */}
      <div className="absolute top-0 right-0 w-96 h-96 bg-primary/20 blur-[120px] rounded-full -mr-48 -mt-48" />
      <div className="absolute bottom-0 left-0 w-96 h-96 bg-primary/10 blur-[100px] rounded-full -ml-48 -mb-48" />

      <div className="relative z-10 max-w-lg text-center">
        {/* Illustration Card */}
        <div className="mb-12 relative">
          <div className="aspect-square w-full max-w-[400px] mx-auto bg-card/40 backdrop-blur-xl rounded-3xl border border-card/50 shadow-2xl flex items-center justify-center overflow-hidden">
            <div className="relative w-full h-full p-8 flex flex-col items-center justify-center">
              <BarChart className="size-30 text-primary/80" />
              <div className="mt-6 space-y-3 w-full">
                <div className="h-2 w-3/4 bg-primary/40 rounded-full mx-auto" />
                <div className="h-2 w-1/2 bg-primary/30 rounded-full mx-auto" />
                <div className="h-2 w-2/3 bg-primary/20 rounded-full mx-auto" />
              </div>
            </div>
          </div>

          {/* Floating Badge */}
          <Card className="absolute -bottom-6 -right-6 p-4 rounded-2xl shadow-xl flex-row items-center gap-3 border">
            <div className="w-10 h-10 rounded-full bg-primary/20 flex items-center justify-center text-primary">
              <Sparkles className="size-5" />
            </div>
            <div className="text-left">
              <p className="text-xs font-bold text-muted-foreground uppercase tracking-widest">VNINDEX</p>
              <p className="text-sm font-bold">+12.4% Growth</p>
            </div>
          </Card>
        </div>

        <h2 className="text-4xl font-semibold leading-tight mb-6">Your AI Copilot for Vietnam Stock Investing</h2>
        <p className="text-lg text-muted-foreground">
          Analyze market trends and optimize your portfolio with AI-powered insights tailored for the Vietnamese market.
        </p>
      </div>
    </div>
  );
}

/* ────────────────────── Main Page ──────────────────────── */

const LoginPage = () => {
  return (
    <div className="flex min-h-screen w-full bg-background text-foreground font-sans antialiased overflow-hidden">
      <LoginForm />
      <HeroPanel />
    </div>
  );
};

export default LoginPage;
