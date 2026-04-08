import { useState } from "react";
import { Trash2, FileText, FileQuestion } from "lucide-react";
import type { Document } from "@/types/document";
import { deleteDocument } from "@/api/document";
import { StatusBadge } from "./StatusBadge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface DocumentListTableProps {
  documents: Document[];
  onRefresh: () => void;
}

export function DocumentListTable({ documents, onRefresh }: DocumentListTableProps) {
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [docToDelete, setDocToDelete] = useState<Document | null>(null);

  const handleDelete = async () => {
    if (!docToDelete) return;
    setDeletingId(docToDelete.id);
    try {
      await deleteDocument(docToDelete.id);
      onRefresh();
    } catch (error) {
      console.error("Failed to delete document:", error);
    } finally {
      setDeletingId(null);
      setDocToDelete(null);
    }
  };

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return "0 Bytes";
    const k = 1024;
    const sizes = ["Bytes", "KB", "MB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  const getFileIcon = (fileType: string) => {
    switch (fileType.toLowerCase()) {
      case "pdf":
      case "docx":
      case "md":
      case "txt":
        return <FileText className="size-4 text-muted-foreground mr-2" />;
      default:
        return <FileQuestion className="size-4 text-muted-foreground mr-2" />;
    }
  };

  if (documents.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center p-10 border rounded-lg border-dashed text-center">
        <FileText className="size-10 text-muted-foreground/50 mb-4" />
        <p className="text-muted-foreground mb-1">Chưa có tài liệu nào.</p>
        <p className="text-sm text-muted-foreground/75">
          Upload tài liệu để tăng cường khả năng trả lời của AI.
        </p>
      </div>
    );
  }

  return (
    <>
      <div className="rounded-lg border overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[300px] font-bold">Document Name</TableHead>
              <TableHead className="font-bold">Size</TableHead>
              <TableHead className="font-bold">Status</TableHead>
              <TableHead className="text-right font-bold">Chunks</TableHead>
              <TableHead className="text-right font-bold w-[100px]">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {documents.map((doc) => (
              <TableRow key={doc.id}>
                <TableCell className="font-medium">
                  <div className="flex items-center">
                    {getFileIcon(doc.file_type)}
                    <span className="truncate max-w-[250px] block" title={doc.name}>
                      {doc.name}
                    </span>
                  </div>
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {formatFileSize(doc.size_bytes)}
                </TableCell>
                <TableCell>
                  <StatusBadge status={doc.status} errorMessage={doc.error_msg} />
                </TableCell>
                <TableCell className="text-right font-mono text-sm text-muted-foreground">
                  {doc.chunk_count}
                </TableCell>
                <TableCell className="text-right">
                  <Button
                    variant="ghost"
                    size="icon"
                    className="text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                    onClick={() => setDocToDelete(doc)}
                  >
                    <Trash2 className="size-4" />
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <Dialog open={!!docToDelete} onOpenChange={(open) => !open && setDocToDelete(null)}>
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>Xóa tài liệu?</DialogTitle>
            <DialogDescription>
              Tài liệu <strong>{docToDelete?.name}</strong> và toàn bộ vector tương ứng sẽ bị xóa. Hành động này không thể hoàn tác.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="mt-4">
            <Button variant="ghost" onClick={() => setDocToDelete(null)} disabled={!!deletingId}>
              Hủy
            </Button>
            <Button variant="destructive" onClick={handleDelete} disabled={!!deletingId}>
              {deletingId ? "Đang xóa..." : "Xóa"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
