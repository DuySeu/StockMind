import { useEffect, useState } from "react";
import { Plus, RotateCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogTrigger,
} from "@/components/ui/dialog";
import type { Document } from "@/types/document";
import { getDocuments } from "@/api/document";
import { DocumentListTable } from "@/components/DocumentListTable";
import { DocumentUploadForm } from "@/components/DocumentUploadForm";
import { useDocumentPolling } from "@/hooks/useDocumentPolling";

const DocumentPage = () => {
  const [documents, setDocuments] = useState<Document[]>([]);
  const [isUploadOpen, setIsUploadOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  const fetchDocuments = async () => {
    try {
      const docs = await getDocuments();
      setDocuments(docs);
    } catch (error) {
      console.error("Failed to fetch documents", error);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchDocuments();
  }, []);

  useDocumentPolling(documents, fetchDocuments);

  const handleUploadSuccess = () => {
    setIsUploadOpen(false);
    fetchDocuments();
  };

  return (
    <div className="w-full flex-1 flex flex-col">
      <header className="w-full border-b border-primary/20 bg-background/80 backdrop-blur-md px-6 lg:px-10 py-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex flex-col items-start gap-2">
            <h2 className="text-2xl tracking-tight">Document Management</h2>
            <p className="text-sm text-muted-foreground">Manage files to power AI Responses</p>
          </div>

          <div className="flex gap-2">
            <Button variant="outline" size="icon" onClick={() => fetchDocuments()}>
              <RotateCw className={`size-4 ${isLoading ? "animate-spin" : ""}`} />
            </Button>
            
            <Dialog open={isUploadOpen} onOpenChange={setIsUploadOpen}>
              <DialogTrigger asChild>
                <Button>
                  <Plus className="mr-2 size-4" />
                  Upload
                </Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                  <DialogTitle>Upload New Document</DialogTitle>
                  <DialogDescription>
                    Supported formats: PDF, DOCX, TXT, MD (Max: 10MB)
                  </DialogDescription>
                </DialogHeader>
                <div className="mt-4">
                  <DocumentUploadForm onSuccess={handleUploadSuccess} />
                </div>
              </DialogContent>
            </Dialog>
          </div>
        </div>
      </header>

      <div className="p-4 max-w-7xl mx-auto w-full">
        {isLoading && documents.length === 0 ? (
          <div className="flex justify-center p-10"><RotateCw className="animate-spin text-muted-foreground size-6" /></div>
        ) : (
          <DocumentListTable documents={documents} onRefresh={fetchDocuments} />
        )}
      </div>
    </div>
  );
};

export default DocumentPage;
