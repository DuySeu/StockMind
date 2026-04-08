export interface ValidationResult {
  valid: boolean;
  error?: string;
}

export function validateDocumentFile(file: File): ValidationResult {
  if (file.size > 10 * 1024 * 1024) {
    return { valid: false, error: "File quá lớn (tối đa 10MB)." };
  }

  const allowedExtensions = [".pdf", ".docx", ".md", ".txt"];
  const extension = (file.name.substring(file.name.lastIndexOf('.')) || '').toLowerCase();
  
  if (!allowedExtensions.includes(extension)) {
    return { valid: false, error: "Định dạng file không hỗ trợ. (Chỉ hỗ trợ: PDF, DOCX, MD, TXT)." };
  }

  return { valid: true };
}
