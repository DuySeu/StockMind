import React, { createContext, useContext, useEffect, useState } from "react";
import { LoadingSpinner } from "../components";
import useIsMobile from "./UseIsMobile";

// Define the context shape
interface ApplicationContextType {
  loading: boolean;
  isOpenSidebar: boolean;
  setIsOpenSidebar: (value: boolean) => void;
  conversations: any[];
  setConversations: (value: any[]) => void;
  currentConversation: IConversation | undefined;
  setCurrentConversation: (value: IConversation | undefined) => void;
}

// Create the context with default values
const ApplicationContext = createContext<ApplicationContextType | undefined>({
  loading: true,
  isOpenSidebar: true,
  setIsOpenSidebar: () => {},
  conversations: [],
  setConversations: () => {},
  currentConversation: undefined,
  setCurrentConversation: () => {},
});

interface IConversation {
  id: string;
  title: string;
}

// Create the provider component
export const ApplicationProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [loading, setLoading] = useState<boolean>(true);
  const [isOpenSidebar, setIsOpenSidebar] = useState<boolean>(true);
  const [currentConversation, setCurrentConversation] = useState<
    IConversation | undefined
  >();
  const [conversations, setConversations] = useState<any[]>([]);
  const isMobile = useIsMobile();

  useEffect(() => {
    setLoading(false);
  }, []);

  useEffect(() => {
    if (isMobile) {
      setIsOpenSidebar(false);
    }
  }, [isMobile]);

  const value: ApplicationContextType = {
    loading,
    isOpenSidebar,
    setIsOpenSidebar,
    conversations,
    setConversations,
    currentConversation,
    setCurrentConversation,
  };

  return (
    <ApplicationContext.Provider value={value}>
      {!loading ? children : <LoadingSpinner />}
    </ApplicationContext.Provider>
  );
};

// Custom hook to use the ApplicationContext
export const useApplicationState = (): ApplicationContextType => {
  const context = useContext(ApplicationContext);
  if (!context) {
    throw new Error(
      "useApplicationState must be used within an ApplicationProvider"
    );
  }
  return context;
};
