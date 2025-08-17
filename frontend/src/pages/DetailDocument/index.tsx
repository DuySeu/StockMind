import { Button, Tag } from "antd";
import { ArchiveIcon, ArrowLeftIcon, DatabaseIcon, FileTextIcon, ImageIcon, ShieldCheckIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { getDocumentById } from "../../api/documents-api";
import { Document, FileDetail } from "../../types/document";
import ComplianceValidation from "./ComplianceValidation";
import DocumentParsing from "./DocumentParsing";
import ExtractDataResult from "./ExtractDataResult";
import { DocumentPreview } from "../../components";
import { getFileType } from "../../utils/DetectFileType";

function DetailDocument() {
  const navigate = useNavigate();
  const location = useLocation();
  const documentId = location.pathname.split("/").pop() as string;
  const document = location.state?.document as Document;

  const [activeTab, setActiveTab] = useState(0);
  const [fileDetail, setFileDetail] = useState<FileDetail | null>(null);

  const fileType = getFileType(document.name);

  useEffect(() => {
    const fetchDocument = async () => {
      try {
        const response = await getDocumentById(documentId);
        setFileDetail(response);
      } catch (error) {
        console.error("Error fetching document:", error);
      }
    };
    fetchDocument();
  }, [documentId]);

  const getQualityColor = (quality: number) => {
    if (!quality) return "gray";
    if (quality >= 80) return "green";
    if (quality >= 60) return "geekblue";
    if (quality < 60) return "red";
    return "gray";
  };

  const getStatusColor = (status: string) => {
    switch (status.toUpperCase()) {
      case "SUCCESS":
        return "bg-green-500";
      case "FAILED":
        return "bg-red-500";
      case "RUNNING":
      case "SCHEDULED":
        return "bg-geekblue-500";
      default:
        return "bg-gray-500";
    }
  };

  const tabItems = [
    { key: "1", label: "Document Parsing", icon: <FileTextIcon /> },
    { key: "2", label: "Compliance Checker", icon: <ShieldCheckIcon /> },
    { key: "3", label: "Extract Data Result", icon: <DatabaseIcon /> },
  ];

  const tabContents = [
    <DocumentParsing document={document} quality={fileDetail?.quality_result || []} />,
    <ComplianceValidation rules_results={fileDetail?.processed_result.validation} />,
    <ExtractDataResult result={fileDetail?.processed_result || null} />,
  ];

  // const handleDownload = () => {
  //   if (fileDetail?.raw_file) {
  //     const url = base64ToDataUrl(fileDetail.raw_file?.base64, fileDetail.raw_file?.type);
  //     const link = window.document.createElement("a");
  //     link.href = url;
  //     link.download = document?.name || "document";
  //     window.document.body.appendChild(link);
  //     link.click();
  //     window.document.body.removeChild(link);
  //   }
  // };

  const overallQuality = useMemo(() => {
    if (!fileDetail?.quality_result) return null;

    // Calculate overall quality for each page
    const pageQualities = fileDetail.quality_result.map((page) => ({
      pageNumber: page.page,
      overallQuality: page.quality_metrics?.overall_quality || 0,
      qualityMetrics: page.quality_metrics || {},
    }));

    // Calculate average overall quality for all pages
    const averageQuality =
      pageQualities.length > 0
        ? pageQualities.reduce((sum: number, page: any) => sum + page.overallQuality, 0) / pageQualities.length
        : 0;

    return averageQuality;
  }, [fileDetail?.quality_result]);

  const renderQuality = () => {
    if (!fileDetail?.quality_result || !overallQuality) return null;
    let criteria: string = "";
    if (overallQuality >= 80) {
      criteria = "Good";
    } else if (overallQuality >= 60) {
      criteria = "Fair";
    } else {
      criteria = "Poor";
    }
    return (
      <Tag color={getQualityColor(overallQuality)}>
        {criteria} ({Math.round(overallQuality)}%)
      </Tag>
    );
  };

  return (
    <>
      <div className="flex items-center justify-between gap-2 mx-2">
        <div className="flex flex-row gap-3 items-center">
          <Button type="text" icon={<ArrowLeftIcon />} onClick={() => navigate("/")} />
          {document ? (
            <>
              <div className="flex items-center gap-1 text-lg font-bold">
                {fileType === "image" ? (
                  <ImageIcon size={20} />
                ) : fileType === "pdf" ? (
                  <FileTextIcon size={20} />
                ) : (
                  <ArchiveIcon size={20} />
                )}
                <p>{document.name}</p>
              </div>
              <div className="flex items-center gap-2 text-sm">
                <p>Transaction ID:</p>
                <p>{document.transaction_id}</p>
              </div>
              {renderQuality()}
              <span className={`rounded-xl px-2 py-1 ${getStatusColor(document.status)} text-white text-sm`}>
                {document.status}
              </span>
            </>
          ) : (
            <p>Loading...</p>
          )}
        </div>
        {/* <div className="flex flex-row gap-2">
          {fileDetail?.raw_file && (
            <Button icon={<DownloadIcon />} onClick={handleDownload}>
              Download
            </Button>
          )}
        </div> */}
      </div>
      <div className="flex gap-2 w-full bg-gray-100 rounded p-1">
        {tabItems.map((tab, idx) => (
          <Button
            key={tab.key}
            variant={activeTab === idx ? "outlined" : "text"}
            color={"default"}
            className={`flex-1 p-1 flex gap-2 items-center justify-center transition-colors`}
            onClick={() => setActiveTab(idx)}
          >
            {tab.icon}
            <span>{tab.label}</span>
          </Button>
        ))}
      </div>
      <div className="flex-1 grid grid-cols-1 md:grid-cols-2 border border-gray-200 rounded-lg">
        <div className="col-span-1 h-full">
          <DocumentPreview raw_file={fileDetail?.raw_file as string} file_name={document?.name || ""} />
        </div>
        <div className="flex flex-col gap-2 border-l border-gray-200 col-span-1">{tabContents[activeTab]}</div>
      </div>
    </>
  );
}

export default DetailDocument;
