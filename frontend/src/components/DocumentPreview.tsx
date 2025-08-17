import { Card, Image, Space, Spin, Tag, Typography } from "antd";
import { ArchiveIcon } from "lucide-react";
import { getFileType } from "../utils/DetectFileType";

const { Text, Title } = Typography;

const DocumentPreview = ({ raw_file, file_name }: { raw_file: string; file_name: string }) => {
  const fileType = getFileType(file_name);

  if (!raw_file) {
    return (
      <div className="flex justify-center items-center h-64">
        <Spin size="large" />
        <Text className="ml-2">Loading document...</Text>
      </div>
    );
  }

  const renderImagePreview = () => (
    <Image src={raw_file} alt={file_name || "Document preview"} fallback="/SampleImage.png" width={"100%"} />
  );

  const renderPDFPreview = () => <iframe src={raw_file as unknown as string} className="w-full h-full" />;

  const renderZipPreview = () => (
    <Card className="max-w-md mx-auto">
      <div className="text-center">
        <ArchiveIcon size={64} className="mx-auto mb-4 text-blue-500" />
        <Title level={4}>ZIP Archive</Title>
        <Text type="secondary" className="block mb-4">
          {file_name}
        </Text>
        <Space direction="vertical" size="middle">
          <Tag color="blue">Archive File</Tag>
        </Space>
      </div>
    </Card>
  );

  const renderLoading = () => (
    <div className="flex justify-center items-center h-64">
      <Spin size="large" />
      <Text className="ml-2">Loading document...</Text>
    </div>
  );

  const renderContent = () => {
    if (fileType === "image") {
      return renderImagePreview();
    } else if (fileType === "pdf") {
      return renderPDFPreview();
    } else if (fileType === "zip") {
      return renderZipPreview();
    } else {
      return renderLoading();
    }
  };

  return <>{renderContent()}</>;
};

export default DocumentPreview;
