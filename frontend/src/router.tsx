import { createBrowserRouter } from "react-router-dom";
import Layout from "@/pages/layout";
import HomePage from "@/pages/HomePage";
import LoginPage from "@/pages/LoginPage";
import SettingPage from "@/pages/SettingPage";
import ErrorPage from "@/pages/ErrorPage";
import ExplorePage from "@/pages/Explore";
import WatchListPage from "@/pages/WatchList";
import LibraryPage from "@/pages/Library";

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
        element: <ExplorePage />,
      },
      {
        path: "watchlist",
        element: <WatchListPage />,
      },
      {
        path: "library",
        element: <LibraryPage />,
      },
    ],
  },
  {
    path: "/login",
    element: <LoginPage />,
    errorElement: <ErrorPage />,
  },
]);
