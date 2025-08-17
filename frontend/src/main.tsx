import { createRoot } from "react-dom/client";
import App from "./App.tsx";
import "./index.css";
import { pdfjs } from "react-pdf";

// Import the worker as a URL for production builds
import pdfWorkerUrl from "pdfjs-dist/build/pdf.worker.min.mjs?url";
import "react-pdf/dist/esm/Page/AnnotationLayer.css";
import "react-pdf/dist/esm/Page/TextLayer.css";

pdfjs.GlobalWorkerOptions.workerSrc = pdfWorkerUrl;

createRoot(document.getElementById("root")!).render(<App />);
