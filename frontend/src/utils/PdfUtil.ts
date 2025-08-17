import { pdfjs } from "react-pdf";

pdfjs.GlobalWorkerOptions.workerSrc = new URL("pdfjs-dist/build/pdf.worker.min.mjs", import.meta.url).toString();

export const convertFileToImages = async (file: File): Promise<string[]> => {
  if (file.type.startsWith("image/")) {
    return [URL.createObjectURL(file)];
  }

  if (file.type === "application/pdf") {
    const url = URL.createObjectURL(file);
    const images: string[] = [];

    try {
      const loadingTask = pdfjs.getDocument(url);
      const pdf = await loadingTask.promise;
      const numPages = pdf.numPages;

      for (let pageNum = 1; pageNum <= numPages; pageNum++) {
        const page = await pdf.getPage(pageNum);
        const scale = 2;
        const viewport = page.getViewport({ scale });

        const canvas = document.createElement("canvas");
        const context = canvas.getContext("2d");
        if (!context) continue;

        canvas.width = viewport.width;
        canvas.height = viewport.height;

        await page.render({ canvasContext: context, viewport }).promise;
        images.push(canvas.toDataURL("image/png"));
      }
    } catch (error) {
      console.error("Error rendering PDF:", error);
    } finally {
      URL.revokeObjectURL(url);
    }

    return images;
  }

  return [];
};
