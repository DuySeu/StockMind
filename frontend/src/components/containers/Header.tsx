import { Check, Ellipsis, PenLine, Upload, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useState } from "react";
import { Input } from "@/components/ui/input";
import { useChatContext } from "@/hooks/context";

interface HeaderProps {
  icon?: React.ReactNode;
  editable?: boolean;
  shouldAnimate?: boolean;
}

const Header = ({ icon, editable, shouldAnimate }: HeaderProps) => {
  const [isEditing, setIsEditing] = useState(false);
  const { title, setTitle } = useChatContext();

  const handleEdit = () => {
    setIsEditing(true);
  };

  const handleSave = () => {
    setTitle(title);
    setIsEditing(false);
  };

  const handleCancel = () => {
    setIsEditing(false);
  };

  return (
    <header className="flex justify-between items-center py-3 px-4">
      <div className="flex items-center gap-2">
        {icon}
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
            } font-semibold text-md text-primary hidden md:block truncate max-w-[150px] md:max-w-[200px] lg:max-w-[300px]`}
          >
            {title}
          </span>
        )}
        {editable && !isEditing && (
          <Button
            variant="secondary"
            size="icon-sm"
            className="bg-card/80 text-card-foreground/60"
            onClick={handleEdit}
          >
            <PenLine />
          </Button>
        )}
      </div>
      <div className="flex items-center gap-2 text-primary">
        <Button variant="outline" size="sm" className="hover:text-card-foreground/60">
          <Upload /> Share
        </Button>
        <Button variant="outline" size="icon-sm" className="hover:text-card-foreground/60">
          <Ellipsis />
        </Button>
      </div>
    </header>
  );
};

export default Header;
