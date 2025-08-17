import { Flex, Progress, Tabs, Tag } from "antd";
import { useEffect, useMemo, useState } from "react";
import { getTemplates } from "../../api/template-api";
import { Template } from "../../types/template";
import { Document, QualityMetrics } from "../../types/document";

const DocumentParsing = ({
  document,
  quality,
}: {
  document: Document | null;
  quality: {
    page: number;
    quality_metrics: QualityMetrics;
  }[];
}) => {
  const [templates, setTemplates] = useState<Template[]>([]);

  useEffect(() => {
    const fetchTemplates = async () => {
      try {
        const response = await getTemplates();
        setTemplates(response);
      } catch (error) {
        console.error("Error fetching templates:", error);
      }
    };
    fetchTemplates();
  }, []);

  const getQualityColor = (quality: number) => {
    if (quality === 0) return "gray";
    if (quality >= 80) return "green";
    if (quality >= 60) return "geekblue";
    if (quality < 60) return "red";
  };

  const renderQuality = () => {
    if (!quality) return <Tag color={getQualityColor(overallQuality)}>{overallQuality}%</Tag>;
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

  const overallQuality = useMemo(() => {
    if (!quality || quality.length === 0) return 0;

    const totalQuality = quality.reduce((sum: number, page: any) => {
      return sum + (page.quality_metrics?.overall_quality || 0);
    }, 0);

    return totalQuality / quality.length;
  }, [quality]);

  const pageTabs = quality.map((page) => ({
    label: `Page ${page.page}`,
    key: page.page.toString(),
    children: (
      <div className="flex flex-col gap-3 p-3 rounded-lg border border-gray-200 bg-white">
        <Flex justify="space-between" align="center" gap={2}>
          <p className="text-sm text-nowrap">Brightness</p>
          <Progress
            percent={Math.round((page.quality_metrics.brightness / 255) * 100)}
            size={[150, 15]}
            percentPosition={{ align: "center", type: "inner" }}
            style={{ width: "fit-content" }}
          />
        </Flex>
        <Flex justify="space-between" align="center" gap={2}>
          <p className="text-sm text-nowrap">Noise Level</p>
          <Progress
            percent={Math.round(page.quality_metrics.background_noise)}
            size={[150, 15]}
            percentPosition={{ align: "center", type: "inner" }}
            style={{ width: "fit-content" }}
          />
        </Flex>
        <Flex justify="space-between" align="center" gap={2}>
          <p className="text-sm text-nowrap">Text Clarity</p>
          <Progress
            percent={Math.round(page.quality_metrics.text_clarity)}
            size={[150, 15]}
            percentPosition={{ align: "center", type: "inner" }}
            style={{ width: "fit-content" }}
          />
        </Flex>
        <Flex justify="space-between" align="center" gap={2}>
          <p className="text-sm text-nowrap">Blur Detection</p>
          {page.quality_metrics.is_blurry ? (
            <span className="text-red-500 rounded-full border border-red-500 w-[150px] text-center text-xs font-bold">
              Blurry
            </span>
          ) : (
            <span className="text-green-500 rounded-full border border-green-500 w-[150px] text-center text-xs font-bold">
              Clear
            </span>
          )}
        </Flex>
        <Flex justify="space-between" align="center" gap={2}>
          <p className="text-sm text-nowrap">Contrast</p>
          {page.quality_metrics.is_low_contrast ? (
            <span className="text-red-500 rounded-full border border-red-500 w-[150px] text-center text-xs font-bold">
              Low
            </span>
          ) : (
            <span className="text-green-500 rounded-full border border-green-500 w-[150px] text-center text-xs font-bold">
              High
            </span>
          )}
        </Flex>
      </div>
    ),
  }));

  return (
    <div className="p-3">
      <p className="text-lg font-bold">Document Parsing</p>
      <p className="text-sm text-gray-500">Configure document type and processing settings</p>
      <div className="mt-2 flex flex-col gap-2">
        <div className="flex flex-row gap-2">
          <span className="font-bold text-sm">Document Type:</span>
          <span className="text-sm">{document?.template_name}</span>
        </div>
        <div className="flex flex-col gap-2 mb-2">
          <span className="font-bold text-sm">Processing Prompt:</span>
          <span className="text-sm p-2 rounded border border-gray-200">
            {templates.find((template) => template.id === document?.template_id)?.prompt}
          </span>
        </div>
      </div>
      <div className="border-t border-gray-200">
        <div className="flex justify-between items-center">
          <p className="text-lg font-bold my-2">Image Quality Assessment</p>
          {renderQuality()}
        </div>
        <Tabs defaultActiveKey="1" size="small" type="card" items={pageTabs} />
      </div>
    </div>
  );
};

export default DocumentParsing;
