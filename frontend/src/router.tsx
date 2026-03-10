import { createBrowserRouter } from "react-router-dom";
import Layout from "@/pages/layout";
import LoginPage from "@/pages/LoginPage";
import ErrorPage from "@/pages/ErrorPage";
import WatchListPage from "@/pages/WatchList";
import MarketResearcherPage from "./pages/MarketResearcherPage";
import ResearchResultPage from "./pages/ResearchResultPage";
import ChatbotPage from "./pages/Chatbot";
import HomePage from "./pages/HomePage";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <HomePage />,
    errorElement: <ErrorPage />,
  },
  {
    path: "/chatbot",
    element: <Layout />,
    errorElement: <ErrorPage />,
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
  },
  {
    path: "/watchlist",
    errorElement: <ErrorPage />,
    element: <WatchListPage />,
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
  },
  {
    path: "/login",
    element: <LoginPage />,
    errorElement: <ErrorPage />,
  },
]);
