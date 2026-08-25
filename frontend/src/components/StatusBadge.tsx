import type { DocumentStatus } from "@/types/document";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Loader2 } from "lucide-react";

interface StatusBadgeProps {
  status: DocumentStatus;
  errorMessage?: string;
  className?: string;
}

export function StatusBadge({ status, errorMessage, className }: StatusBadgeProps) {
  switch (status) {
    case "pending":
      return (
        <Badge variant="secondary" className={className}>
          Pending
        </Badge>
      );
    case "processing":
      return (
        <Badge variant="secondary" className={`flex items-center gap-1 ${className || ""}`}>
          <Loader2 className="size-3 animate-spin" />
          Processing
        </Badge>
      );
    case "ready":
      return (
        <Badge variant="secondary" className={`bg-status-ok-bg text-status-ok ${className || ""}`}>
          Ready
        </Badge>
      );
    case "failed":
      return (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Badge variant="destructive" className={`cursor-help ${className || ""}`}>
                Failed
              </Badge>
            </TooltipTrigger>
            <TooltipContent className="max-w-[300px] text-sm">
              <p>{errorMessage || "Processing failed unconditionally"}</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      );
    default:
      return (
        <Badge variant="outline" className={className}>
          {status}
        </Badge>
      );
  }
}
