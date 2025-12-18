import api from "./index";

export interface ChatMessage {
  content: string;
  session_id?: string;
}

export interface ChatResponse {
  type: "thinking_delta" | "text_delta" | "complete";
  data?: {
    thinking?: string;
    text?: string;
    session_id?: string;
  };
}

export const chatWithLLM = async (
  content: string,
  sessionId: string | undefined,
  onMessage: (data: ChatResponse) => void,
  onError: (error: any) => void
) => {
  try {
    const response = await fetch(`${api.defaults.baseURL}/chat`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ content, session_id: sessionId }),
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    if (!response.body) {
      throw new Error("Response body is null");
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder("utf-8");
    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n\n");
      buffer = lines.pop() || "";

      for (const line of lines) {
        if (line.startsWith("data: ")) {
          const data = line.slice(6);
          try {
            // Handle potential multiple JSON objects in one line if backend flushes weirdly,
            // though \n\n split should handle standard SSE.
            // The backend sends: fmt.Fprintf(w, "data: %s\n\n", data)
            const parsed = JSON.parse(data) as ChatResponse;
            onMessage(parsed);
          } catch (e) {
            console.error("Error parsing SSE data:", e);
          }
        }
      }
    }
  } catch (error) {
    onError(error);
  }
};

export const getMessages = async (sessionId: string): Promise<any[]> => {
  const response = await api.get(`/sessions/${sessionId}`);
  return response.data;
};
