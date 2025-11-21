import SideBar from "@/components/containers/SideBar";
import { chatWithLLM } from "@/api/chat";
import { Button } from "@/components/ui/button";
import { SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { Input } from "@/components/ui/input";
import { useForm, type FieldValues } from "react-hook-form";
import { Form, FormControl, FormDescription, FormField, FormItem } from "@/components/ui/form";
import { Calendar, Home, Inbox, Search, Send, Settings, Sun } from "lucide-react";
import MessageList from "@/components/containers/MessageList";
import { useState } from "react";

type Message = {
  role: "user" | "assistant";
  content: Record<string, unknown>[];
};

const sidebarItems = [
  {
    title: "Home",
    url: "#",
    icon: Home,
  },
  {
    title: "Inbox",
    url: "#",
    icon: Inbox,
  },
  {
    title: "Calendar",
    url: "#",
    icon: Calendar,
  },
  {
    title: "Search",
    url: "#",
    icon: Search,
  },
  {
    title: "Settings",
    url: "#",
    icon: Settings,
  },
];

const HomePage = () => {
  const [conversationId] = useState<string | null>(null);
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
      <SideBar items={sidebarItems} />
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
