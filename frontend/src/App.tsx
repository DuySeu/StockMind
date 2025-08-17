import { ConfigProvider } from "antd";
import { BrowserRouter, Route, Routes } from "react-router";
import { ErrorBoundary } from "./components";
import { ROUTES } from "./constants";
import { ApplicationProvider } from "./hooks/ApplicationContext";
import { ComplianceChecker, CreateTemplatePage, DetailDocument, HomePage } from "./pages";
import { WrapperLayout } from "./layout";
import { NotificationProvider } from "./hooks/Notification";

function App() {
  return (
    <BrowserRouter>
      <ConfigProvider
        theme={{
          token: {
            colorPrimary: "#f9ae3b",
            fontFamily: "Manrope",
          },
        }}
      >
        <ApplicationProvider>
          <NotificationProvider>
            <Routes>
              <Route
                path={ROUTES.CREATE_TEMPLATE}
                element={
                  <ErrorBoundary>
                    <WrapperLayout title="Create Template">
                      <CreateTemplatePage />
                    </WrapperLayout>
                  </ErrorBoundary>
                }
              />
              <Route
                path={ROUTES.COMPLIANCE_CHECKER}
                element={
                  <ErrorBoundary>
                    <WrapperLayout title="Compliance Checker">
                      <ComplianceChecker />
                    </WrapperLayout>
                  </ErrorBoundary>
                }
              />
              <Route
                path={ROUTES.HOME}
                element={
                  <ErrorBoundary>
                    <WrapperLayout title="Intelligent Document Processing">
                      <HomePage />
                    </WrapperLayout>
                  </ErrorBoundary>
                }
              />
              <Route
                path={ROUTES.DETAIL_DOCUMENT + "/:id"}
                element={
                  <ErrorBoundary>
                    <WrapperLayout title="Detail Document">
                      <DetailDocument />
                    </WrapperLayout>
                  </ErrorBoundary>
                }
              />
            </Routes>
          </NotificationProvider>
        </ApplicationProvider>
      </ConfigProvider>
    </BrowserRouter>
  );
}

export default App;
