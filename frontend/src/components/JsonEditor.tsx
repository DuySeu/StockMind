import { Button, Input, message } from "antd";
import { useRef } from "react";

const JsonEditor = ({
  value,
  onChange,
  placeholder,
}: {
  value?: string;
  onChange?: (value: string) => void;
  placeholder?: string;
}) => {
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    const textarea = e.currentTarget;
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const text = textarea.value;

    if (e.key === "{") {
      e.preventDefault();
      const newText = text.slice(0, start) + "{\n\t\n}" + text.slice(end);
      textarea.value = newText;
      textarea.selectionStart = start + 2;
      textarea.selectionEnd = start + 2;
      onChange?.(newText);
    } else if (e.key === '"') {
      e.preventDefault();
      const newText = text.slice(0, start) + '""' + text.slice(end);
      textarea.value = newText;
      textarea.selectionStart = start + 1;
      textarea.selectionEnd = start + 1;
      onChange?.(newText);
    } else if (e.key === "Enter") {
      const beforeCursor = text.slice(0, start);
      //   const afterCursor = text.slice(end);

      // Check if we're inside braces
      const openBraces = (beforeCursor.match(/\{/g) || []).length;
      const closeBraces = (beforeCursor.match(/\}/g) || []).length;

      if (openBraces > closeBraces) {
        e.preventDefault();
        const indent = "\t".repeat(openBraces - closeBraces);
        const newText = text.slice(0, start) + "\n" + indent + text.slice(end);
        textarea.value = newText;
        textarea.selectionStart = start + 1 + indent.length;
        textarea.selectionEnd = start + 1 + indent.length;
        onChange?.(newText);
      }
    }
  };

  const formatJson = () => {
    try {
      const parsed = JSON.parse(value || "{}");
      const formatted = JSON.stringify(parsed, null, 2);
      onChange?.(formatted);
    } catch (error) {
      console.log("error", error);
      message.error("Invalid JSON format");
    }
  };

  return (
    <div style={{ position: "relative" }}>
      <Input.TextArea
        ref={textareaRef}
        value={value}
        onChange={(e) => onChange?.(e.target.value)}
        onKeyDown={handleKeyDown}
        autoSize={{ minRows: 5, maxRows: 5 }}
        placeholder={placeholder}
        style={{ fontFamily: "monospace", fontSize: "12px" }}
      />
      <Button size="small" onClick={formatJson} style={{ position: "absolute", top: 4, right: 4 }}>
        Format
      </Button>
    </div>
  );
};

export default JsonEditor;
