import React, { useEffect } from "react";
import Sidebar from "./Sidebar";
import styles from "./styles.module.scss";
import { Typography } from "antd";
import { useApplicationState } from "../hooks/ApplicationContext";
import { PanelRightOpenIcon, PanelRightCloseIcon } from "lucide-react";
import { ROUTES } from "../constants";

function WrapperLayout({ children, title }: { children: React.ReactNode; title: string }) {
  const { isOpenSidebar, setIsOpenSidebar } = useApplicationState();

  useEffect(() => {
    const isDetailDocumentRoute = location.pathname.startsWith(ROUTES.DETAIL_DOCUMENT);

    if (isDetailDocumentRoute) {
      // Close sidebar when on detail document route
      setIsOpenSidebar(false);
    } else {
      // Open sidebar when not on detail document route
      setIsOpenSidebar(true);
    }
  }, [location.pathname, setIsOpenSidebar]);

  return (
    <div className="flex min-h-screen w-full overflow-hidden">
      <Sidebar />
      <div className="h-screen w-full overflow-y-auto flex-1">
        <div className="bg-gradient-to-b from-white to-primary/20 p-5 overflow-y-auto">
          <div className={styles.WrapperContainer}>
            <div className="flex flex-row gap-2 items-center">
              <button onClick={() => setIsOpenSidebar(!isOpenSidebar)}>
                {isOpenSidebar ? <PanelRightOpenIcon /> : <PanelRightCloseIcon />}
              </button>
              <Typography.Title className="!text-primary !text-[24px] !font-extrabold !mb-0">{title}</Typography.Title>
            </div>
            {children}
          </div>
        </div>
      </div>
    </div>
  );
}

export default WrapperLayout;
