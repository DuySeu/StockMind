import jsPDF from "jspdf";

export interface PDFConfig {
  margin: number;
  lineHeight: number;
  pageWidth: number;
  pageHeight: number;
  contentWidth: number;
}

export class PDFGenerator {
  doc: jsPDF;
  y: number;
  config: PDFConfig;

  constructor(orientation: "p" | "l" = "p") {
    this.doc = new jsPDF({ orientation, unit: "mm", format: "a4" });
    const pageWidth = this.doc.internal.pageSize.getWidth();
    const pageHeight = this.doc.internal.pageSize.getHeight();
    const margin = 15;

    this.config = {
      margin,
      lineHeight: 6,
      pageWidth,
      pageHeight,
      contentWidth: pageWidth - margin * 2 - 2,
    };
    this.y = margin;
  }

  sanitize(text: string): string {
    return text
      .replace(/[\u2014\u2013]/g, "-") // em/en dash -> hyphen
      .replace(/[\u2018\u2019]/g, "'") // smart single quotes
      .replace(/[\u201C\u201D]/g, '"') // smart double quotes
      .replace(/\u2026/g, "...") // ellipsis
      .replace(/\u2022/g, "-") // bullet
      .replace(/[\u00A0]/g, " ") // non-breaking space
      .replace(/[^\x00-\xFF]/g, ""); // strip anything outside Latin-1
  }

  checkPageBreak(requiredSpace: number) {
    if (this.y + requiredSpace > this.config.pageHeight - this.config.margin) {
      this.doc.addPage();
      this.y = this.config.margin;
    }
  }

  setFont(size: number, style: string, r: number, g: number, b: number) {
    this.doc.setFontSize(size);
    this.doc.setFont("helvetica", style);
    this.doc.setTextColor(r, g, b);
  }

  addSectionTitle(title: string) {
    this.checkPageBreak(16);
    this.y += 2;
    this.setFont(13, "bold", 30, 64, 175);
    this.doc.text(this.sanitize(title), this.config.margin, this.y);
    this.y += 2;
    this.doc.setDrawColor(30, 64, 175);
    this.doc.setLineWidth(0.4);
    this.doc.line(this.config.margin, this.y, this.config.pageWidth - this.config.margin, this.y);
    this.y += 7;
  }

  addWrappedText(text: string, fontSize = 10, fontStyle: string = "normal") {
    const clean = this.sanitize(text);
    this.setFont(fontSize, fontStyle, 50, 50, 50);
    const lines: string[] = this.doc.splitTextToSize(clean, this.config.contentWidth);
    for (const line of lines) {
      this.checkPageBreak(this.config.lineHeight + 1);
      this.setFont(fontSize, fontStyle, 50, 50, 50);
      this.doc.text(line, this.config.margin, this.y);
      this.y += this.config.lineHeight;
    }
    this.y += 3;
  }

  addHeader(ticker: string, companyName: string) {
    this.doc.setFillColor(30, 64, 175);
    this.doc.rect(0, 0, this.config.pageWidth, 36, "F");

    this.setFont(20, "bold", 255, 255, 255);
    this.doc.text(this.sanitize(`${ticker} - Research Report`), this.config.margin, 16);

    this.setFont(11, "normal", 220, 220, 255);
    this.doc.text(this.sanitize(companyName), this.config.margin, 24);

    this.setFont(9, "normal", 200, 200, 240);
    this.doc.text(`Generated: ${new Date().toLocaleDateString()}`, this.config.margin, 31);
    this.y = 44;
  }

  addMetricsBoard(marketCap?: string, peRatio?: string) {
    if (!marketCap && !peRatio) return;

    this.checkPageBreak(18);
    this.doc.setFillColor(235, 240, 255);
    this.doc.roundedRect(this.config.margin, this.y, this.config.contentWidth, 14, 2, 2, "F");

    if (marketCap) {
      this.setFont(10, "bold", 40, 40, 40);
      this.doc.text(this.sanitize(`Market Cap: ${marketCap}`), this.config.margin + 5, this.y + 9);
    }
    if (peRatio) {
      this.setFont(10, "bold", 40, 40, 40);
      this.doc.text(
        this.sanitize(`P/E Ratio: ${peRatio}`),
        this.config.margin + this.config.contentWidth / 2,
        this.y + 9,
      );
    }
    this.y += 22;
  }

  addSourceLinks(sources: any[]) {
    if (!sources?.length) return;

    this.addSectionTitle("Research Sources");
    sources.forEach((src: any, i: number) => {
      const url = typeof src === "string" ? src : src.url;
      if (url) {
        this.setFont(9, "normal", 30, 64, 175);
        const prefix = `${i + 1}. `;
        const cleanUrl = this.sanitize(url);
        const lines: string[] = this.doc.splitTextToSize(prefix + cleanUrl, this.config.contentWidth);

        lines.forEach((line) => {
          this.checkPageBreak(this.config.lineHeight + 1);
          this.setFont(9, "normal", 30, 64, 175);
          this.doc.textWithLink(line, this.config.margin, this.y, { url });
          this.y += this.config.lineHeight;
        });
        this.y += 2;
      }
    });
  }

  addFooter() {
    const totalPages = this.doc.getNumberOfPages();
    for (let p = 1; p <= totalPages; p++) {
      this.doc.setPage(p);
      this.setFont(8, "italic", 150, 150, 150);
      this.doc.text(`StockMind Research - Page ${p} of ${totalPages}`, this.config.margin, this.config.pageHeight - 8);
    }
  }

  save(filename: string) {
    this.doc.save(filename);
  }
}
