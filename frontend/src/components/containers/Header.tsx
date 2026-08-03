import { Check, Ellipsis, Loader2, MessageSquareText, PenLine, Upload, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useEffect, useRef, useState } from "react";
import { Input } from "@/components/ui/input";
import { toast } from "sonner";

/** Matches the server's rune cap in UpdateSessionTitleHandler. */
const MAX_TITLE_LENGTH = 120;

interface HeaderProps {
  shouldAnimate?: boolean;
  title: string;
  /** Only a saved conversation can be renamed — a new chat has no row yet. */
  canRename?: boolean;
  /** Persists the new title. Rejecting keeps the editor open with the draft intact. */
  onRename?: (title: string) => Promise<void>;
}

const Header = ({ shouldAnimate, title, canRename = false, onRename }: HeaderProps) => {
  const [isEditing, setIsEditing] = useState(false);
  // The draft is local: editing used to write straight through to the parent's
  // title, so Cancel had nothing to restore and left the edit in place.
  const [draft, setDraft] = useState(title);
  const [isSaving, setIsSaving] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  // A title that changes underneath us (rename from the sidebar, session
  // switch) should not be clobbered by a stale draft.
  useEffect(() => {
    if (!isEditing) setDraft(title);
  }, [title, isEditing]);

  // Leaving a conversation closes the editor rather than carrying it over.
  useEffect(() => {
    if (!canRename) setIsEditing(false);
  }, [canRename]);

  const startEditing = () => {
    setDraft(title);
    setIsEditing(true);
  };

  const cancelEditing = () => {
    setDraft(title);
    setIsEditing(false);
  };

  const trimmed = draft.trim();
  const canSave = trimmed.length > 0 && trimmed !== title && !isSaving;

  const save = async () => {
    if (isSaving) return;
    if (!trimmed) {
      // Silently reverting would look like the save worked.
      toast.error("Title cannot be empty.");
      inputRef.current?.focus();
      return;
    }
    if (trimmed === title) {
      setIsEditing(false);
      return;
    }
    if (!onRename) {
      setIsEditing(false);
      return;
    }

    setIsSaving(true);
    try {
      await onRename(trimmed);
      setIsEditing(false);
    } catch (error) {
      console.error("Failed to rename conversation", error);
      toast.error("Could not rename this conversation. Please try again.");
      inputRef.current?.focus();
    } finally {
      setIsSaving(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    // The input lives inside the page, not a form, so Enter needs wiring.
    // `isComposing` keeps a Telex/VNI syllable commit from submitting.
    if (e.key === "Enter" && !e.nativeEvent.isComposing) {
      e.preventDefault();
      void save();
    } else if (e.key === "Escape") {
      e.preventDefault();
      cancelEditing();
    }
  };

  return (
    <header className="flex shrink-0 items-center gap-3 border-b border-border bg-muted/50 px-4 py-2.5">
      <div className="flex min-w-0 flex-1 items-center gap-2">
        <MessageSquareText className="size-5 shrink-0 text-muted-foreground" aria-hidden="true" />
        {isEditing ? (
          <>
            <Input
              ref={inputRef}
              type="text"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={handleKeyDown}
              onBlur={(e) => {
                // Blurring onto Save/Cancel must not pre-empt their click.
                if (e.currentTarget.parentElement?.contains(e.relatedTarget as Node)) return;
                cancelEditing();
              }}
              aria-label="Conversation title"
              maxLength={MAX_TITLE_LENGTH}
              disabled={isSaving}
              autoFocus
              className="h-8 max-w-sm"
            />
            <Button
              variant="default"
              size="icon-sm"
              onClick={save}
              disabled={!canSave}
              aria-label="Save title"
            >
              {isSaving ? <Loader2 className="animate-spin" aria-hidden="true" /> : <Check aria-hidden="true" />}
            </Button>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={cancelEditing}
              disabled={isSaving}
              aria-label="Cancel renaming"
            >
              <X aria-hidden="true" />
            </Button>
          </>
        ) : (
          <>
            <h1
              className={`truncate text-sm font-semibold text-foreground ${shouldAnimate ? "animate-fade-right" : ""}`}
            >
              {title}
            </h1>
            {/* Gated on "is there a conversation", not on the title text — the
                pencil used to appear for any title other than the literal
                "StockMind", including on a brand-new chat with nothing to save. */}
            {canRename && (
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={startEditing}
                aria-label="Rename conversation"
                className="shrink-0 text-muted-foreground hover:text-foreground"
              >
                <PenLine aria-hidden="true" />
              </Button>
            )}
          </>
        )}
      </div>

      {/* Secondary actions stay quiet: the send button is the only filled
          control on this screen, so it keeps its emphasis. */}
      <div className="flex shrink-0 items-center gap-1">
        <Button variant="ghost" size="sm" className="text-muted-foreground hover:text-foreground">
          <Upload aria-hidden="true" />
          <span className="hidden sm:inline">Share</span>
        </Button>
        <Button
          variant="ghost"
          size="icon-sm"
          aria-label="More options"
          className="text-muted-foreground hover:text-foreground"
        >
          <Ellipsis aria-hidden="true" />
        </Button>
      </div>
    </header>
  );
};

export default Header;
