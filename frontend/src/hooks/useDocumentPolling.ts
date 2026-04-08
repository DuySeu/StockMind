import { useEffect, useRef } from "react";
import type { Document } from "@/types/document";

export function useDocumentPolling(documents: Document[], refreshCallback: () => Promise<void>) {
  const shouldPoll = documents.some(d => d.status === "pending" || d.status === "processing");
  const savedCallback = useRef(refreshCallback);

  useEffect(() => {
    savedCallback.current = refreshCallback;
  }, [refreshCallback]);

  useEffect(() => {
    if (!shouldPoll) return;

    const tick = () => {
      savedCallback.current();
    };

    const id = setInterval(tick, 3000);
    return () => clearInterval(id);
  }, [shouldPoll]);
}
