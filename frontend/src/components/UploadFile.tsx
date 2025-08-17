import { Button, Select, Upload } from "antd";
import { ArchiveIcon, CloudUploadIcon, FileTextIcon, ImageIcon, XIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { getTemplates } from "../api/template-api";
import { useNotificationService } from "../hooks/Notification";
import { Template } from "../types/template";
import { createDocument } from "../api/documents-api";

const UploadFile = ({
  isModalOpen,
  onUploadSuccess,
}: {
  isModalOpen: (value: boolean) => void;
  onUploadSuccess: () => void;
}) => {
  const { createErrorNotification, createSuccessNotification } = useNotificationService();

  const [files, setFiles] = useState<any[]>([]);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [selectedTemplates, setSelectedTemplates] = useState<Record<string, string>>({});
  const [isUploading, setIsUploading] = useState(false);
  const templateIds: string[] = [];

  useEffect(() => {
    const fetchTemplates = async () => {
      try {
        const response = await getTemplates();
        setTemplates(response);
      } catch (error) {
        console.error("Error fetching templates:", error);
        createErrorNotification("Failed to fetch templates");
      }
    };
    fetchTemplates();
  }, [createErrorNotification]);

  const onHandleChangeFile = (info: any) => {
    const MAX_FILE_SIZE = 50 * 1024 * 1024; // 50MB
    const validFiles = info.fileList.filter((file: any) => {
      if (file.size > MAX_FILE_SIZE) {
        createErrorNotification(`${file.name} exceeds 50MB limit and has been removed`);
        return false;
      }
      return true;
    });

    setFiles(validFiles);
    const currentFileIds = validFiles.map((file: any) => file.uid);
    const updatedTemplates = { ...selectedTemplates };
    Object.keys(updatedTemplates).forEach((fileId) => {
      if (!currentFileIds.includes(fileId)) {
        delete updatedTemplates[fileId];
      }
    });
    setSelectedTemplates(updatedTemplates);
  };
  const onDelete = (file: any) => {
    const newFiles = files.filter((item) => item.uid !== file.uid);
    setFiles(newFiles);
    const updatedTemplates = { ...selectedTemplates };
    delete updatedTemplates[file.uid];
    setSelectedTemplates(updatedTemplates);
  };

  // Handle template selection change
  const onTemplateChange = (fileId: string, templateId: string) => {
    setSelectedTemplates((prev) => ({
      ...prev,
      [fileId]: templateId,
    }));
  };

  // Validate that all files have templates selected
  const isUploadValid = () => {
    if (files.length === 0) return false;
    return files.every((file) => selectedTemplates[file.uid]);
  };

  const onUpload = async () => {
    setIsUploading(true);
    const formData = new FormData();
    try {
      for (const file of files) {
        formData.append("files", file.originFileObj);
        templateIds.push(selectedTemplates[file.uid]);
      }
      formData.append("template_ids", templateIds.join(","));
      const response = await createDocument(formData);
      createSuccessNotification(response.message);
    } catch (error) {
      console.error("Error uploading files:", error);
      createErrorNotification("Error uploading files");
      setIsUploading(false);
    } finally {
      isModalOpen(false);
      setIsUploading(false);
      setFiles([]);
      setSelectedTemplates({});
      onUploadSuccess();
      templateIds.length = 0;
    }
  };

  return (
    <div className="w-full max-w-[640px] m-auto flex flex-col gap-8">
      {files.length < 5 && (
        <Upload.Dragger
          multiple
          name="file"
          maxCount={5}
          onChange={onHandleChangeFile}
          beforeUpload={() => false}
          accept=".pdf, image/*, .zip"
          showUploadList={false}
          fileList={files}
        >
          <p className="ant-upload-drag-icon flex justify-center text-primary">
            <CloudUploadIcon size={60} />
          </p>
          <p className="ant-upload-text">Click or drag file to this area to upload</p>
          <div className="ant-upload-hint flex items-center justify-center gap-2 mb-2">
            <span className="rounded-full px-2 py-1 text-sm bg-primary/70 text-white">Max 5 files</span>
            <span className="rounded-full px-2 py-1 text-sm bg-primary/70 text-white">Max 10MB each</span>
          </div>
          <div className="flex items-center justify-center gap-4 text-gray-500">
            <div className="flex items-center gap-1">
              <FileTextIcon />
              <p>PDF</p>
            </div>
            <div className="flex items-center gap-1">
              <ImageIcon />
              <p>Image</p>
            </div>
            <div className="flex items-center gap-1">
              <ArchiveIcon />
              <p>Zip</p>
            </div>
          </div>
        </Upload.Dragger>
      )}

      {/* File Preview Section */}
      {files.length > 0 && (
        <div className="border border-gray-200 rounded-lg p-4">
          <div className="flex justify-between items-center mb-4">
            <p className="text-lg font-bold">Selected Files</p>
            <div className="flex items-center gap-4">
              <span className="border border-primary rounded-full px-2 text-sm text-primary">{`${files.length}/5 Files`}</span>
              <Button
                onClick={() => {
                  setFiles([]);
                  setSelectedTemplates({});
                }}
                color="danger"
                variant="filled"
                disabled={isUploading}
              >
                Clear All
              </Button>
            </div>
          </div>
          <div className="space-y-2">
            {files.map((file: any) => (
              <div key={file.uid} className="flex items-center justify-between p-4 bg-gray-50 rounded">
                <div className="flex items-center gap-2">
                  <FileTextIcon className="text-primary" size={32} />
                  <div className="flex flex-col gap-1">
                    <p>{file.name}</p>
                    <span className="text-xs font-bold text-primary border border-gray-200 rounded-full px-2 w-fit bg-primary/15">
                      {(file.size / 1024 / 1024).toFixed(2)} MB
                    </span>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <Select
                    placeholder="Select Template"
                    options={templates.map((template) => ({ label: template.name, value: template.id }))}
                    style={{ width: 160 }}
                    value={selectedTemplates[file.uid] || undefined}
                    onChange={(value) => onTemplateChange(file.uid, value)}
                    status={!selectedTemplates[file.uid] ? "error" : undefined}
                  />
                  <Button
                    icon={<XIcon />}
                    onClick={() => onDelete(file)}
                    variant="text"
                    color="danger"
                    disabled={isUploading}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <Button type="primary" htmlType="submit" disabled={!isUploadValid()} onClick={onUpload} loading={isUploading}>
        Upload {files.length} Document{files.length > 1 ? "s" : ""} for Processing
      </Button>
    </div>
  );
};

export default UploadFile;
