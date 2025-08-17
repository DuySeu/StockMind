export function getFileType(filename: string): string {
  const parts = filename.split(".");
  const fileType = parts.length > 1 ? parts.pop()!.toLowerCase() : "";
  switch (fileType) {
    case "pdf":
      return "pdf";
    case "zip":
      return "zip";
    default:
      return "image";
  }
}
