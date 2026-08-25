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
import { Skeleton } from "@/components/ui/skeleton";
import { useDocumentPolling } from "@/hooks/useDocumentPolling";

// Render the knowledge-base document list with upload and polling refresh
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
      <header className="w-full border-b border-border bg-background/85 backdrop-blur-md">
        <div className="mx-auto flex max-w-7xl flex-wrap items-end justify-between gap-4 px-4 py-5 sm:px-6 lg:px-8">
          <div className="flex flex-col gap-1">
            <h1 className="text-xl font-bold tracking-tight">Knowledge base</h1>
            <p className="text-sm text-muted-foreground">Documents the assistant can cite when it answers.</p>
          </div>

          <div className="flex gap-2">
            <Button variant="outline" size="icon" onClick={fetchDocuments} aria-label="Refresh documents">
              <RotateCw className={isLoading ? "animate-spin" : ""} aria-hidden="true" />
            </Button>

            <Dialog open={isUploadOpen} onOpenChange={setIsUploadOpen}>
              <DialogTrigger asChild>
                {/* mr-2 fought the button's own gap-2 and pushed the label off-centre. */}
                <Button>
                  <Plus aria-hidden="true" />
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

      <div className="mx-auto w-full max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        {isLoading && documents.length === 0 ? (
          /* Skeleton rows in the table's own shape, not a spinner in empty space. */
          <div className="overflow-hidden rounded-lg border border-border">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex items-center gap-4 border-b border-border p-4 last:border-b-0">
                <Skeleton className="h-4 flex-1" />
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-5 w-16 rounded-full" />
              </div>
            ))}
          </div>
        ) : (
          <DocumentListTable documents={documents} onRefresh={fetchDocuments} />
        )}
      </div>
    </div>
  );
};

export default DocumentPage;
