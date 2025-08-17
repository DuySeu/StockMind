import { RawText } from "../../components";

function ExtractDataResult({ result }: { result: any }) {
  console.log("ExtractDataResult result", result);
  return (
    <>
      <div className="flex flex-col gap-2 mt-4">
        <p className="text-lg font-bold px-3">Extracted Data</p>
        <p className="text-sm text-gray-500 px-3">Review and edit extracted information · Modified 0 Fields</p>
        <RawText data={result.raw_text || "No data"} />
      </div>
      <div className="border-t border-gray-200">
        <p className="text-lg font-bold p-3">Json Data</p>
        <pre
          style={{
            backgroundColor: "#f5f5f5",
            padding: 8,
            overflow: "auto",
            maxHeight: 300,
            fontSize: 14,
          }}
        >
          {JSON.stringify(result.json_data, null, 2)}
        </pre>
      </div>
    </>
  );
}

export default ExtractDataResult;
