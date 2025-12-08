import { chatWithLLM } from "@/api/chat";
import { getSession } from "@/api/sessions";
import MessageList from "@/components/containers/MessageList";
import SideBar from "@/components/containers/SideBar";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormDescription, FormField, FormItem } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { Send, Sun } from "lucide-react";
import { useEffect, useState } from "react";
import { useForm, type FieldValues } from "react-hook-form";

type Message = {
  role: "user" | "assistant";
  content: Record<string, unknown>[];
};

const HomePage = () => {
  const [conversationId] = useState<string | null>(null);
  const [isFirstMessage, setIsFirstMessage] = useState<boolean>(true);
  const [messages, setMessages] = useState<Message[]>([]);
  const [sessions, setSessions] = useState<any[]>([]);

  const form = useForm({
    defaultValues: {
      input: "",
    },
  });

  useEffect(() => {
    getSession().then((data) => {
      setSessions(data);
      console.log(data);
    });
  }, []);

  const onSubmit = async (data: FieldValues) => {
    console.log(data);
    form.reset();

    if (!conversationId) {
      setIsFirstMessage(true);
    }

    setMessages([...messages, { role: "user", content: [{ type: "text", text: data.input.trim() }] }]);

    const assistantIndex = messages.length + 1;
    setMessages((prev) => [...prev, { role: "assistant", content: [] }]);

    await chatWithLLM(
      data.input.trim(),
      conversationId || undefined,
      (data) => {
        switch (data.type) {
          case "thinking_delta": {
            const delta = data.data?.thinking ?? "";
            setMessages((prev) => {
              const updated = [...prev];
              const currentMessage = updated[assistantIndex] || { role: "assistant", content: [] };
              const newContent = [...(currentMessage.content || [])];

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
              updated[assistantIndex] = { ...currentMessage, content: newContent };
              return updated;
            });
            break;
          }
          case "text_delta": {
            const delta = data.data?.text ?? "";
            setMessages((prev) => {
              const updated = [...prev];
              const currentMessage = updated[assistantIndex] || { role: "assistant", content: [] };
              const newContent = [...(currentMessage.content || [])];

              let idx = newContent.findIndex((c) => c.type === "text");
              if (idx === -1) {
                newContent.push({ type: "text", text: "" });
                idx = newContent.length - 1;
              }
              const block = newContent[idx];
              newContent[idx] = { ...block, text: (block.text ?? "") + delta };
              updated[assistantIndex] = { ...currentMessage, content: newContent };
              return updated;
            });
            break;
          }
          case "complete": {
            // if (data.data) {
            //   setConversationId(data.data);
            // }
            setMessages((prev) => {
              const updated = [...prev];
              const currentMessage = updated[assistantIndex] || { role: "assistant", content: [] };
              const newContent = [...(currentMessage.content || [])];

              let idx = newContent.findIndex((c) => c.type === "thinking");
              if (idx !== -1) {
                const block = newContent[idx];
                newContent[idx] = { ...block, is_open: false };
                updated[assistantIndex] = { ...currentMessage, content: newContent };
              }
              return updated;
            });
            break;
          }
        }
      },
      (error) => {
        console.error("Error sending message:", error);
        setMessages((prev) => {
          const updated = [...prev];
          updated[assistantIndex] = {
            role: "assistant",
            content: [{ type: "text", text: "Error sending message" }],
          };
          return updated;
        });
      }
    );

    if (isFirstMessage) {
      setIsFirstMessage(false);
    }
  };

  return (
    <SidebarProvider>
      <SideBar items={sessions} />
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
