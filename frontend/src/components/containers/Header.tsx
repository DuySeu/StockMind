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
    <header className="flex justify-between items-center py-3 px-4">
      <div className="flex items-center gap-2">
        <MessageSquareText className="text-primary-foreground w-6 h-6" />
        {isEditing ? (
          <>
            <Input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full text-primary"
            />
            <Button variant="secondary" size="icon-sm" onClick={handleSave}>
              <Check />
            </Button>
            <Button variant="destructive" size="icon-sm" onClick={handleCancel}>
              <X />
            </Button>
          </>
        ) : (
          <span
            className={`${
              shouldAnimate ? "animate-fade-right" : ""
            } font-semibold text-md text-primary-foreground hidden md:block truncate max-w-[150px] md:max-w-[200px] lg:max-w-[300px]`}
          >
            {title}
          </span>
        )}
        {title !== "StockMind" && !isEditing && (
          <Button variant="secondary" size="icon-sm" onClick={handleEdit}>
            <PenLine />
          </Button>
        )}
      </div>
      <div className="flex items-center gap-2 text-primary">
        <Button size="sm" className="hover:text-card-foreground/60">
          <Upload /> Share
        </Button>
        <Button size="icon-sm" className="hover:text-card-foreground/60">
          <Ellipsis />
        </Button>
      </div>
    </header>
  );
};

export default Header;
