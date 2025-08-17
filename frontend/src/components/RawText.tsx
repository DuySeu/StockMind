import { Pagination } from "antd";
import { useEffect, useMemo, useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeRaw from "rehype-raw";

function RawText({ data }: { data: any }) {
  const [currentPage, setCurrentPage] = useState(1);

  const pages = useMemo(() => {
    if (!data || typeof data !== "string") {
      return ["No raw text available"];
    }
    return data.split("=== PAGE BREAK ===");
  }, [data]);

  useEffect(() => {
    setCurrentPage(1);
  }, [data]);

  return (
    <div className="bg-[#f5f5f5]">
      <ReactMarkdown
        className="react-markdown-custom prose max-w-none p-2 max-h-[500px] overflow-y-auto"
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeRaw]}
      >
        {pages[currentPage - 1]}
      </ReactMarkdown>
      {pages.length > 1 && (
        <Pagination
          current={currentPage}
          total={pages.length * 10}
          pageSize={10}
          onChange={setCurrentPage}
          style={{ marginTop: 16, textAlign: "center" }}
          showSizeChanger={false}
          className="!justify-center"
        />
      )}
    </div>
  );
}

export default RawText;
