import { useNavigate } from "react-router-dom";
import TemplateIcon from "../assets/icons/TemplateIcon";
import { ROUTES } from "../constants";
import { useApplicationState } from "../hooks/ApplicationContext";
import styles from "./styles.module.scss";
import ReportIcon from "../assets/icons/ReportIcon";
import { ActivityIcon } from "lucide-react";

function Sidebar() {
  const { isOpenSidebar } = useApplicationState();
  const navigate = useNavigate();

  return (
    <div
      className={`${styles.wrapperSidebar} ${
        isOpenSidebar ? "w-[320px]" : "w-0"
      } min-h-screen max-h-screen bg-white border-none flex flex-col transition-all ease-in	duration-300 overflow-hidden relative pt-8`}
    >
      {/* <button className="absolute top-3 right-0 m-1 p-2 w-fit z-10" onClick={() => setIsOpenSidebar(!isOpenSidebar)}>
        <CloseIcon />
      </button> */}
      <div
        className="flex m-auto cursor-pointer"
        onClick={() => {
          navigate(ROUTES.HOME);
        }}
      >
        <img src="../../images/techx-logo.png" alt="TechX Logo" className="object-cover" />
      </div>

      <nav className="flex-1 px-2 overflow-y-auto mt-8 py-2 gap-2 flex flex-col">
        <button
          className={`w-full flex items-center gap-5 px-5 py-3 text-[14px] font-semibold text-[#000000CC] rounded transition-all duration-400 ease-in-out hover:bg-gray-100 cursor-pointer ${
            location.pathname === ROUTES.HOME ? "bg-[#FF7F001A]" : "bg-transparent"
          }`}
          onClick={() => navigate(ROUTES.HOME)}
        >
          <ActivityIcon />
          <span className="whitespace-nowrap">Monitoring</span>
        </button>
        <button
          className={`w-full flex items-center gap-5 px-5 py-3 text-[14px] font-semibold text-[#000000CC] rounded transition-all duration-400 ease-in-out hover:bg-gray-100 cursor-pointer ${
            location.pathname === ROUTES.CREATE_TEMPLATE || location.pathname === ROUTES.DETAIL_DOCUMENT
              ? "bg-[#FF7F001A]"
              : "bg-transparent"
          }`}
          onClick={() => navigate(ROUTES.CREATE_TEMPLATE)}
        >
          <TemplateIcon />
          <span className="whitespace-nowrap">Templates</span>
        </button>
        <button
          className={`w-full flex items-center gap-5 px-5 py-3 text-[14px] font-semibold text-[#000000CC] rounded transition-all duration-400 ease-in-out hover:bg-gray-100 cursor-pointer ${
            location.pathname === ROUTES.COMPLIANCE_CHECKER ? "bg-[#FF7F001A]" : "bg-transparent"
          }`}
          onClick={() => navigate(ROUTES.COMPLIANCE_CHECKER)}
        >
          <ReportIcon />
          <span className="whitespace-nowrap">Compliance Checker</span>
        </button>
      </nav>
    </div>
  );
}

export default Sidebar;
