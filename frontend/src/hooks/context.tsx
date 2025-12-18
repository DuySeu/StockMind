import { createContext, useContext, useState, type ReactNode } from "react";

interface ChatContextType {
  title: string;
  setTitle: (title: string) => void;
}

const ChatContext = createContext<ChatContextType | undefined>(undefined);

export const ChatProvider = ({ children }: { children: ReactNode }) => {
  const [title, setTitle] = useState<string>("StockMind");

  return <ChatContext.Provider value={{ title, setTitle }}>{children}</ChatContext.Provider>;
};

export const useChatContext = () => {
  const context = useContext(ChatContext);
  if (context === undefined) {
    throw new Error("useChatContext must be used within a ChatProvider");
  }
  return context;
};
