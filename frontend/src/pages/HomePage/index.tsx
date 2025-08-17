import { Button, Input, Modal, Popconfirm, Table, Tag } from "antd";
import { LoaderCircleIcon, PlusIcon, RefreshCcwIcon, RotateCwIcon, SearchIcon, TriangleAlertIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { executeFlow, getDocuments } from "../../api/documents-api";
import { UploadFile } from "../../components";
import { ROUTES } from "../../constants";
import { Document } from "../../types/document";

function HomePage() {
  const navigate = useNavigate();

  const [searchText, setSearchText] = useState("");
  const [documents, setDocuments] = useState<Document[]>([]);
  const [loading, setLoading] = useState(true);
  const [isModalOpen, setIsModalOpen] = useState(false);

  useEffect(() => {
    fetchDocuments();
  }, []);

  const fetchDocuments = async () => {
    try {
      setLoading(true);
      const response = await getDocuments();
      setDocuments(response);
    } catch (error) {
      console.error(error);
      setDocuments([]);
    } finally {
      setLoading(false);
    }
  };

  // const getAverageConfidence = () => {
  //   if (!Array.isArray(documents) || documents.length === 0) return 0;
  //   const totalConfidence = documents
  //     .filter((doc) => doc.confidence !== null)
  //     .reduce((acc, doc) => acc + parseFloat(doc.confidence || "0"), 0);
  //   return Math.round((totalConfidence / documents.length) * 100) || 0;
  // };

  const getAverageQuality = () => {
    if (!Array.isArray(documents) || documents.length === 0) return 0;
    const totalQuality = documents.reduce((acc, doc) => {
      if (doc.quality?.toUpperCase() === "GOOD") {
        return acc + 1;
      }
      return acc;
    }, 0);
    return ((totalQuality / documents.length) * 100).toFixed(2) || 0;
  };

  const getUnprocessedDocuments = () => {
    if (!Array.isArray(documents)) return "0";
    const unprocessedStatuses = ["PENDING", "RUNNING", "FAILED", "SCHEDULED"];
    return documents.filter((doc) => unprocessedStatuses.includes(doc.status.toUpperCase())).length.toString();
  };

  // Ensure documents is an array before using reduce
  const groupedData = Array.isArray(documents)
    ? documents.reduce((acc, doc) => {
        const parent = acc.find((item) => item.transaction_id === doc.transaction_id);
        if (parent) {
          parent.children = [...(parent.children || []), { ...doc, children: undefined }];
        } else {
          acc.push({ ...doc });
        }
        return acc;
      }, [] as Document[])
    : [];

  const filteredData = groupedData.filter((doc) => {
    const isMatch =
      doc.name.toUpperCase().includes(searchText.toUpperCase()) ||
      doc.transaction_id.toUpperCase().includes(searchText.toUpperCase());
    if (doc.children) {
      return (
        isMatch ||
        doc.children.some(
          (child) =>
            child.name.toUpperCase().includes(searchText.toUpperCase()) ||
            child.transaction_id.toUpperCase().includes(searchText.toUpperCase())
        )
      );
    }
    return isMatch;
  });

  const columns = [
    {
      title: "Transaction ID",
      dataIndex: "transaction_id",
      key: "transaction_id",
    },
    {
      title: "File Name",
      dataIndex: "name",
      key: "name",
    },
    {
      title: "Type",
      dataIndex: "template_name",
      key: "template_name",
    },
    {
      title: "Status",
      dataIndex: "status",
      key: "status",
      render: (status: string) => {
        let color;
        if (status.toUpperCase() === "FAILED") color = "red";
        if (status.toUpperCase() === "SUCCESS") color = "green";
        if (status.toUpperCase() === "RUNNING") color = "geekblue";
        if (status.toUpperCase() === "PENDING") color = "yellow";
        if (status.toUpperCase() === "SCHEDULED") color = "geekblue";
        return (
          <Tag color={color} key={status} bordered={false}>
            {status.toUpperCase()}
          </Tag>
        );
      },
    },
    // {
    //   title: "Confidence",
    //   dataIndex: "confidence",
    //   key: "confidence",
    //   render: (confidence: string) => {
    //     if (!confidence) return <p>-</p>;
    //     const percent = Math.round(parseFloat(confidence) * 100);
    //     return <p>{percent}%</p>;
    //   },
    // },
    {
      title: "Quality",
      dataIndex: "quality",
      key: "quality",
      render: (quality: string) => {
        if (!quality) return <p>-</p>;
        return (
          <Tag color={getQualityColor(quality)} key={quality}>
            {quality.toUpperCase()}
          </Tag>
        );
      },
    },
    {
      title: "Action",
      dataIndex: "status",
      key: "status",
      render: (_: any, record: Document) => {
        return renderActionButton(record);
      },
    },
  ];

  const renderActionButton = (record: Document) => {
    console.log(record);
    switch (record.status.toUpperCase()) {
      case "SUCCESS":
        return (
          <div onClick={(e) => e.stopPropagation()}>
            <Popconfirm
              title="Reprocessing document"
              description="Are you sure to reprocess this document?"
              onConfirm={() => {
                executeFlow(record.flow_id);
                fetchDocuments();
              }}
              okText="Yes"
              cancelText="No"
            >
              <Button type="text" icon={<RotateCwIcon size={18} color="green" />} />
            </Popconfirm>
          </div>
        );
      case "FAILED":
        return (
          <div onClick={(e) => e.stopPropagation()}>
            <Popconfirm
              title="Reprocessing document"
              description="Are you sure to reprocess this document?"
              onConfirm={() => {
                executeFlow(record.flow_id);
                fetchDocuments();
              }}
              okText="Yes"
              cancelText="No"
            >
              <Button type="text" icon={<RotateCwIcon size={18} color="red" />} danger />
            </Popconfirm>
          </div>
        );
      case "RUNNING":
      case "SCHEDULED":
        return <Button type="text" icon={<LoaderCircleIcon className="animate-spin" size={18} />} />;
      case "PENDING":
        return <Button type="text" icon={<TriangleAlertIcon size={18} color="orange" />} />;
      default:
        return <Button type="text" icon={<TriangleAlertIcon size={18} />} />;
    }
  };

  const getQualityColor = (quality: string) => {
    if (!quality) return "gray";
    if (quality.toUpperCase() === "GOOD") return "green";
    if (quality.toUpperCase() === "FAIR") return "geekblue";
    if (quality.toUpperCase() === "POOR") return "red";
    return "gray";
  };

  return (
    <>
      <div className="bg-gray-100 flex flex-row gap-3 justify-between rounded-lg p-3">
        <div className="rounded-lg w-1/2 p-8 bg-white shadow-md">
          <p className="font-bold">Unprocessed</p>
          <p className="font-semibold text-3xl">{getUnprocessedDocuments()} Documents</p>
        </div>
        {/* <div className="rounded-lg w-1/3 p-8 bg-white shadow-md">
          <p className="font-bold">Average Confidence</p>
          <p className="font-semibold text-3xl">{getAverageConfidence()}%</p>
        </div> */}
        <div className="rounded-lg w-1/2 p-8 bg-white shadow-md">
          <p className="font-bold">Quality</p>
          <p className="font-semibold text-3xl">{getAverageQuality()}% Good</p>
        </div>
      </div>
      <div className="flex flex-col gap-4">
        <div className="flex flex-row gap-4 items-center justify-between">
          <p className="!text-primary !text-[24px] !font-extrabold !mb-0">All Documents</p>
          <div className="flex flex-row gap-2">
            <Button variant="outlined" icon={<RefreshCcwIcon size={18} />} onClick={fetchDocuments} />
            <Input
              placeholder="Search by transaction ID or file name"
              className="!w-[300px]"
              prefix={<SearchIcon size={18} />}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
            />
            <Button type="primary" icon={<PlusIcon />} onClick={() => setIsModalOpen(true)}>
              Upload Document
            </Button>
          </div>
        </div>
        <Table<Document>
          dataSource={filteredData}
          columns={columns}
          loading={loading}
          expandable={{ defaultExpandAllRows: true }}
          onRow={(record) => ({
            onClick:
              record.status.toUpperCase() === "SUCCESS"
                ? () => navigate(`${ROUTES.DETAIL_DOCUMENT}/${record.id}`, { state: { document: record } })
                : undefined,
            style: record.status.toUpperCase() === "SUCCESS" ? { cursor: "pointer" } : { cursor: "default" },
          })}
          rowKey="id"
          pagination={{
            pageSize: 10,
            showSizeChanger: false,
            showQuickJumper: false,
            showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} transactions`,
          }}
        />
      </div>
      <Modal
        title="Upload Document"
        open={isModalOpen}
        onCancel={() => setIsModalOpen(false)}
        footer={null}
        width={1000}
      >
        <UploadFile isModalOpen={setIsModalOpen} onUploadSuccess={fetchDocuments} />
      </Modal>
    </>
  );
}

export default HomePage;
