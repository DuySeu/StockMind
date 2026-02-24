import { createBrowserRouter } from "react-router-dom";
import Layout from "@/pages/layout";
import HomePage from "@/pages/HomePage";
import LoginPage from "@/pages/LoginPage";
import SettingPage from "@/pages/SettingPage";
import ErrorPage from "@/pages/ErrorPage";
import WatchListPage from "@/pages/WatchList";
import PendingPage from "./pages/PendingPage";
import MarketResearcherPage from "./pages/MarketResearcherPage";
import ResearchResultPage from "./pages/ResearchResultPage";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <Layout />,
    errorElement: <ErrorPage />,
    children: [
      {
        index: true,
        element: <HomePage />,
      },
      {
        path: "c/:id",
        element: <HomePage />,
      },
      {
        path: "setting",
        element: <SettingPage />,
      },
      {
        path: "explore",
        element: <PendingPage />,
      },
      {
        path: "watchlist",
        element: <WatchListPage />,
      },
      {
        path: "library",
        element: <PendingPage />,
      },
      {
        path: "research",
        element: <MarketResearcherPage />,
      },
      {
        path: "research/:id",
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
