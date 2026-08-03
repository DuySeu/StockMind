import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Form, FormControl, FormField, FormItem } from "@/components/ui/form";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ArrowUp, AudioLines, FileText, Image, Paperclip, Square, X, Zap } from "lucide-react";
import { useRef, useState } from "react";
import { useForm, type FieldValues } from "react-hook-form";
import { toast } from "sonner";
import { Toggle } from "../ui/toggle";

const MAX_FILE_SIZE = 10 * 1024 * 1024;
/** Roughly 7 lines of text before the composer starts scrolling internally. */
const MAX_COMPOSER_HEIGHT = 200;

interface ChatInputProps {
  onSend: (message: string, attachment: File | null, maxMode: boolean) => void;
  isStreaming?: boolean;
  /** Aborts the in-flight stream. Omit to hide the stop affordance. */
  onStop?: () => void;
}

const ChatInput = ({ onSend, isStreaming = false, onStop }: ChatInputProps) => {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const [attachment, setAttachment] = useState<File | null>(null);
  const [maxMode, setMaxMode] = useState(false);

  const form = useForm({
    defaultValues: {
      input: "",
    },
  });

  const handleFileClick = (accept: string) => {
    if (fileInputRef.current) {
      fileInputRef.current.accept = accept;
      fileInputRef.current.click();
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      if (file.size > MAX_FILE_SIZE) {
        toast.error("File is too large. Max size is 10MB.");
      } else {
        setAttachment(file);
      }
    }
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  // Grown by hand rather than with `field-sizing: content`: the primitive ships
  // that class, but Firefox and older Safari ignore it and the composer would
  // silently stop growing there.
  const resizeToContent = (el: HTMLTextAreaElement) => {
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, MAX_COMPOSER_HEIGHT)}px`;
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Shift+Enter is the newline. `isComposing` keeps an IME commit — Telex and
    // VNI both end a syllable with Enter — from firing the send.
    if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
      e.preventDefault();
      form.handleSubmit(handleSubmit)();
    }
  };

  const handleSubmit = (data: FieldValues) => {
    const userInput = data.input.trim();
    if (!userInput) return;

    const fileToSend = attachment;
    form.reset();
    setAttachment(null);
    // Drop the inline height so the box collapses back to one row.
    if (textareaRef.current) textareaRef.current.style.height = "";
    onSend(userInput, fileToSend, maxMode);
  };

  return (
    <div className="surface-solid absolute bottom-0 left-0 w-full p-4 md:px-6">
      <div className="relative mx-auto w-full max-w-3xl">
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit(handleSubmit)}
            className="glass-raised flex flex-col gap-1 rounded-xl p-2 focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/30"
          >
            {attachment && (
              <div className="mb-1 flex w-fit items-center gap-2 rounded-lg border border-border bg-muted px-2.5 py-1.5">
                <div className="flex items-center gap-2 text-sm">
                  {attachment?.type.startsWith("image/") ? (
                    <Image className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
                  ) : (
                    <FileText className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
                  )}
                  <span className="max-w-[180px] truncate">{attachment?.name}</span>
                </div>
                <button
                  type="button"
                  aria-label={`Remove attachment ${attachment?.name ?? ""}`}
                  className="cursor-pointer rounded-full p-1 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                  onClick={() => setAttachment(null)}
                >
                  <X className="size-3.5" aria-hidden="true" />
                </button>
              </div>
            )}

            <FormField
              control={form.control}
              name="input"
              render={({ field }) => (
                <FormItem className="flex-1">
                  <FormControl>
                    <Textarea
                      className="field-sizing-fixed min-h-11 w-full resize-none overflow-y-auto border-none bg-transparent py-3 text-base shadow-none selection:bg-primary selection:text-primary-foreground placeholder:text-muted-foreground focus-visible:ring-0 dark:bg-transparent"
                      style={{ maxHeight: MAX_COMPOSER_HEIGHT }}
                      placeholder="Ask me anything about Vietnam stocks…"
                      aria-label="Message"
                      autoComplete="off"
                      rows={1}
                      disabled={isStreaming}
                      {...field}
                      ref={(el) => {
                        field.ref(el);
                        textareaRef.current = el;
                      }}
                      onChange={(e) => {
                        field.onChange(e);
                        resizeToContent(e.currentTarget);
                      }}
                      onKeyDown={handleKeyDown}
                    />
                  </FormControl>
                </FormItem>
              )}
            />

            <div className="flex w-full items-center justify-between gap-1">
              <div className="flex items-center gap-1">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button
                      type="button"
                      aria-label="Attach a file"
                      className="flex size-10 shrink-0 cursor-pointer items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring disabled:opacity-50"
                    >
                      <Paperclip className="size-5" aria-hidden="true" />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent className="w-44 rounded-lg" align="start">
                    <DropdownMenuItem onClick={() => handleFileClick("*/*")} className="cursor-pointer">
                      <FileText className="h-4 w-4 mr-2" />
                      Upload file
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => handleFileClick("image/*")} className="cursor-pointer">
                      <Image className="h-4 w-4 mr-2" />
                      Upload photo
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
                <Toggle
                  aria-label="Max mode"
                  variant="outline"
                  pressed={maxMode}
                  onPressedChange={setMaxMode}
                  disabled={isStreaming}
                  className="cursor-pointer"
                >
                  <Zap className="size-4" aria-hidden="true" />
                  {maxMode && "Max mode"}
                </Toggle>
              </div>

              <div className="flex items-center gap-1">
                {form.watch("input")?.trim() ? (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button
                        type="submit"
                        disabled={isStreaming || !form.watch("input")?.trim()}
                        className="flex size-10 shrink-0 cursor-pointer items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-xs transition-all hover:bg-primary/90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring active:scale-95 disabled:cursor-not-allowed disabled:opacity-40"
                      >
                        <ArrowUp className="size-5" aria-hidden="true" />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>Send message</p>
                    </TooltipContent>
                  </Tooltip>
                ) : !isStreaming ? (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button
                        type="button"
                        aria-label="Dictate"
                        className="hidden size-10 shrink-0 cursor-pointer items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring md:flex"
                      >
                        <AudioLines className="size-5" aria-hidden="true" />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>Dictate</p>
                    </TooltipContent>
                  </Tooltip>
                ) : (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button
                        type="button"
                        onClick={onStop}
                        disabled={!onStop}
                        aria-label="Stop generating"
                        className="flex size-10 shrink-0 cursor-pointer items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-xs transition-all hover:bg-primary/90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring active:scale-95 disabled:cursor-not-allowed disabled:opacity-40"
                      >
                        <Square className="size-4 fill-current" aria-hidden="true" />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>Stop generating</p>
                    </TooltipContent>
                  </Tooltip>
                )}
              </div>
            </div>
            <input type="file" ref={fileInputRef} className="hidden" onChange={handleFileChange} />
          </form>
        </Form>
        <p className="mt-2.5 text-center text-xs text-muted-foreground">
          "StockMind can make mistakes. Verify important financial info."
        </p>
      </div>
    </div>
  );
};

export default ChatInput;
