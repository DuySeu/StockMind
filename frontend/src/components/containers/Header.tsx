import { Check, Ellipsis, MessageSquareText, PenLine, Upload, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useState } from "react";
import { Input } from "@/components/ui/input";

interface HeaderProps {
  shouldAnimate?: boolean;
  title: string;
  setTitle: (title: string) => void;
}

const Header = ({ shouldAnimate, title, setTitle }: HeaderProps) => {
  const [isEditing, setIsEditing] = useState(false);

  const handleEdit = () => {
    setIsEditing(true);
  };

  const handleSave = () => {
    setIsEditing(false);
  };

  const handleCancel = () => {
    setIsEditing(false);
  };

  return (
    <header className="flex shrink-0 items-center gap-3 border-b border-border bg-muted/50 px-4 py-2.5">
      <div className="flex min-w-0 flex-1 items-center gap-2">
        <MessageSquareText className="size-5 shrink-0 text-muted-foreground" aria-hidden="true" />
        {isEditing ? (
          <>
            <Input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              aria-label="Conversation title"
              autoFocus
              className="h-8 max-w-sm"
            />
            <Button variant="default" size="icon-sm" onClick={handleSave} aria-label="Save title">
              <Check aria-hidden="true" />
            </Button>
            <Button variant="ghost" size="icon-sm" onClick={handleCancel} aria-label="Cancel renaming">
              <X aria-hidden="true" />
            </Button>
          </>
        ) : (
          <h1
            className={`truncate text-sm font-semibold text-foreground ${shouldAnimate ? "animate-fade-right" : ""}`}
          >
            {title}
          </h1>
        )}
        {title !== "StockMind" && !isEditing && (
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={handleEdit}
            aria-label="Rename conversation"
            className="shrink-0 text-muted-foreground hover:text-foreground"
          >
            <PenLine aria-hidden="true" />
          </Button>
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
