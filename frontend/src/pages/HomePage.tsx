import SideBar from "@/components/containers/SideBar";
import { Button } from "@/components/ui/button";
import { SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { Input } from "@/components/ui/input";
import { useForm, type FieldValues } from "react-hook-form";
import { Form, FormControl, FormDescription, FormField, FormItem } from "@/components/ui/form";
import { Send, Sun } from "lucide-react";
import MessageList from "@/components/containers/MessageList";
import { useState } from "react";

type Message = {
  role: "user" | "assistant";
  content: Record<string, unknown>[];
};

const HomePage = () => {
  const [conversationId, setConversationId] = useState<string | null>(null);
  const [isFirstMessage, setIsFirstMessage] = useState<boolean>(true);
  const [messages, setMessages] = useState<Message[]>([]);

  const form = useForm({
    defaultValues: {
      input: "",
    },
  });

  const onSubmit = async (data: FieldValues) => {
    console.log(data);
    form.reset();

    if (!conversationId) {
      setIsFirstMessage(true);
    }

    setMessages([...messages, { role: "user", content: [{ type: "text", text: data.input.trim() }] }]);

    const assistantIndex = messages.length;
    setMessages([...messages, { role: "assistant", content: [] }]);

    try {
      const response = await fetch("/api/chat", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "text/event-stream",
        },

        body: JSON.stringify({ message: data.input.trim(), conversationId: conversationId }),
      });

      if (!response.ok || !response.body) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const chunk = decoder.decode(value, { stream: true });
        buffer += chunk;

        let newlineIndex;
        while ((newlineIndex = buffer.indexOf("\n")) !== -1) {
          const line = buffer.slice(0, newlineIndex).trimEnd();
          buffer = buffer.slice(newlineIndex + 1);

          if (line.startsWith("data: ")) {
            try {
              const data = JSON.parse(line.slice(6));
              switch (data.type) {
                case "thinking_delta": {
                  const delta = data.data?.thinking ?? "";
                  const current = messages[assistantIndex].content ?? [];
                  const updated = [...messages];
                  const newContent = [...current];
                  let idx = newContent.findIndex((c) => c.type === "thinking");
                  if (idx === -1) {
                    newContent.push({
                      type: "thinking",
                      thinking: "",
                      signature: "",
                      is_open: true,
                    });
                    idx = newContent.length - 1;
                  }
                  const block = newContent[idx];
                  newContent[idx] = { ...block, thinking: (block.thinking ?? "") + delta };
                  updated[assistantIndex] = { role: "assistant", content: newContent };
                  setMessages(updated);
                  break;
                }
                case "text_delta": {
                  const delta = data.data?.text ?? "";
                  const current = messages[assistantIndex].content ?? [];
                  const updated = [...messages];
                  const newContent = [...current];
                  let idx = newContent.findIndex((c) => c.type === "text");
                  if (idx === -1) {
                    newContent.push({ type: "text", text: "" });
                    idx = newContent.length - 1;
                  }
                  const block = newContent[idx];
                  newContent[idx] = { ...block, text: (block.text ?? "") + delta };
                  updated[assistantIndex] = { role: "assistant", content: newContent };
                  setMessages(updated);
                  break;
                }
                case "message_stop":
                case "complete": {
                  if (data.data) {
                    setConversationId(data.data);
                  }
                  const current = messages[assistantIndex].content ?? [];
                  const updated = [...messages];
                  const newContent = [...current];
                  let idx = newContent.findIndex((c) => c.type === "thinking");
                  if (idx === -1) {
                    newContent.push({
                      type: "thinking",
                      thinking: "",
                      signature: "",
                      is_open: false,
                    });
                    idx = newContent.length - 1;
                  }
                  const block = newContent[idx];
                  newContent[idx] = { ...block, is_open: false };
                  updated[assistantIndex] = { role: "assistant", content: newContent };
                  setMessages(updated);
                  break;
                }
                case "error": {
                  const error = data.error ?? "Unknown error";
                  const updated = [...messages];
                  updated[assistantIndex] = {
                    role: "assistant",
                    content: [{ type: "text", text: `Error: ${error}` }],
                  };
                  setMessages(updated);
                  break;
                }
                default:
                  break;
              }
            } catch (error) {
              console.error("Error parsing JSON:", error);
            }
          }
        }
      }
    } catch (error) {
      console.error("Error parsing JSON:", error);
      const updated = [...messages];
      updated[assistantIndex] = {
        role: "assistant",
        content: [{ type: "text", text: "Error sending message" }],
      };
      setMessages(updated);
    } finally {
      if (isFirstMessage) {
        setIsFirstMessage(false);
      }
    }
  };

  return (
    <SidebarProvider>
      <SideBar />
      <main className="w-full flex flex-col">
        <header className="flex justify-between items-center bg-secondary p-2">
          <SidebarTrigger />
          <Button variant="outline" size="icon" aria-label="Theme">
            <Sun />
          </Button>
        </header>
        <div className="flex flex-col flex-1 p-2">
          <MessageList messages={messages} />
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="flex gap-2 py-2">
              <FormField
                control={form.control}
                name="input"
                render={({ field }) => (
                  <FormItem className="flex-1">
                    <FormControl>
                      <Input placeholder="Send your message" {...field} />
                    </FormControl>
                  </FormItem>
                )}
              />
              <Button variant="outline" size="icon" aria-label="Submit" disabled={form.watch("input") === ""}>
                <Send />
              </Button>
            </form>
            <FormDescription className="text-center">StockMind AI Assistant Powered by DuySeu</FormDescription>
          </Form>
        </div>
      </main>
    </SidebarProvider>
  );
};

export default HomePage;
