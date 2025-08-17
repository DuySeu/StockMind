import { notification } from "antd";
import React, { createContext, useContext } from "react";

interface NotificationContextType {
  createInfoNotification: (
    description: string | React.ReactNode,
    message?: string
  ) => void;
  createSuccessNotification: (
    description: string | React.ReactNode,
    message?: string
  ) => void;
  createWarningNotification: (
    description: string | React.ReactNode,
    message?: string
  ) => void;
  createErrorNotification: (
    description: string | React.ReactNode,
    message?: string
  ) => void;
}

const NotificationContext = createContext<NotificationContextType | null>(null);

export const NotificationProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [api, contextHolder] = notification.useNotification();

  const createNotification =
    (type: "info" | "success" | "warning" | "error") =>
    (description: string | React.ReactNode, message?: string) => {
      api[type]({
        className: `${type}-notification font-primary top-14`,
        message: description || "",
        description: message,
        duration: 4,
      });
    };

  return (
    <NotificationContext.Provider
      value={{
        createInfoNotification: createNotification("info"),
        createSuccessNotification: createNotification("success"),
        createWarningNotification: createNotification("warning"),
        createErrorNotification: createNotification("error"),
      }}
    >
      {contextHolder}
      {children}
    </NotificationContext.Provider>
  );
};

export const useNotificationService = () => {
  const context = useContext(NotificationContext);
  if (!context) {
    throw new Error(
      "useNotificationService must be used within a NotificationProvider"
    );
  }
  return context;
};
