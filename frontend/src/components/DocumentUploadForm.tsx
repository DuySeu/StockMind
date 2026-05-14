import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Upload, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { validateDocumentFile } from "@/lib/validation";
import { uploadDocument } from "@/api/document";

const formSchema = z.object({
  file: z
    .instanceof(File, {
      message: "Vui lòng chọn một file",
    })
    .superRefine((file, ctx) => {
      const result = validateDocumentFile(file);
      if (!result.valid) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: result.error,
        });
      }
    }),
});

type FormValues = z.infer<typeof formSchema>;

export function DocumentUploadForm({ onSuccess }: { onSuccess?: () => void }) {
  const [isUploading, setIsUploading] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
  });

  const onSubmit = async (data: FormValues) => {
    setIsUploading(true);
    setServerError(null);
    try {
      await uploadDocument(data.file);
      form.reset();
      onSuccess?.();
    } catch (err: any) {
      setServerError(err?.response?.data?.message || err.message || "Upload failed");
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
        {serverError && <div className="p-3 bg-destructive/10 text-destructive text-sm rounded-md">{serverError}</div>}

        <FormField
          control={form.control}
          name="file"
          render={({ field: { value, onChange, ...field } }) => (
            <FormItem>
              <FormLabel>Tài liệu</FormLabel>
              <FormControl>
                <div className="flex items-center gap-4">
                  <Input
                    type="file"
                    accept=".pdf,.docx,.md,.txt"
                    onChange={(e) => {
                      const file = e.target.files?.[0];
                      if (file) {
                        onChange(file);
                      }
                    }}
                    {...field}
                  />
                </div>
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <Button type="submit" disabled={isUploading} className="w-full">
          {isUploading ? (
            <>
              <Loader2 className="animate-spin mr-2 size-4" />
              Đang upload...
            </>
          ) : (
            <>
              <Upload className="mr-2 size-4" />
              Upload Document
            </>
          )}
        </Button>
      </form>
    </Form>
  );
}
