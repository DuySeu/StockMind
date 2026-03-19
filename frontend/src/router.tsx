import { createBrowserRouter } from "react-router-dom";
import LoginPage from "@/pages/LoginPage";
import ErrorPage from "@/pages/ErrorPage";
import WatchListPage from "@/pages/WatchList";
import MarketResearcherPage from "./pages/MarketResearcherPage";
import ResearchResultPage from "./pages/ResearchResultPage";
import ChatbotPage from "./pages/Chatbot";
import HomePage from "./pages/HomePage";
import PendingPage from "./pages/PendingPage";
import { MainLayout } from "./components/layout/MainLayout";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <MainLayout />,
    errorElement: <ErrorPage />,
    children: [
      {
        index: true,
        element: <HomePage />,
      },
      {
        path: "watchlist",
        element: <WatchListPage />,
      },
      {
        path: "research",
        children: [
          {
            index: true,
            element: <MarketResearcherPage />,
          },
          {
            path: ":id",
            element: <ResearchResultPage />,
          },
        ],
      },
      {
        path: "news",
        element: <PendingPage />,
      },
    ],
  },
  {
    path: "/c",
    children: [
      {
        index: true,
        element: <ChatbotPage />,
      },
      {
        path: ":id",
        element: <ChatbotPage />,
      },
    ],
    errorElement: <ErrorPage />,
  },
  {
    path: "/login",
    element: <LoginPage />,
    errorElement: <ErrorPage />,
  },
]);
