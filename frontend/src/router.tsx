import { createBrowserRouter } from "react-router-dom";
import LoginPage from "@/pages/LoginPage";
import ErrorPage from "@/pages/ErrorPage";
import WatchListPage from "@/pages/WatchList";
import MarketResearcherPage from "./pages/MarketResearcherPage";
import ResearchResultPage from "./pages/ResearchResultPage";
import ChatbotPage from "./pages/Chatbot";
import HomePage from "./pages/HomePage";
import PendingPage from "./pages/PendingPage";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <HomePage />,
    errorElement: <ErrorPage />,
  },
  {
    path: "/chat",
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
    path: "/watchlist",
    element: <WatchListPage />,
    errorElement: <ErrorPage />,
  },
  {
    path: "/research",
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
    errorElement: <ErrorPage />,
  },
  {
    path: "/news",
    element: <PendingPage />,
    errorElement: <ErrorPage />,
  },
  {
    path: "/login",
    element: <LoginPage />,
    errorElement: <ErrorPage />,
  },
]);
